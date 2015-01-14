package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const (
	contentMediaType = "application/vnd.git-media"
	metaMediaType    = contentMediaType + "+json"
)

var (
	logger = log.New(os.Stdout, "harbour:", log.LstdFlags)
)

type Meta struct {
	Oid   string           `json:"oid"`
	Size  int64            `json:"size"`
	Links map[string]*link `json:"_links"`
}

type apiMeta struct {
	Oid       string `json:"oid"`
	Size      int64  `json:"size"`
	Writeable bool   `json:"writeable"`
	existing  bool   `json:"-"`
}

type link struct {
	Href   string            `json:"href"`
	Header map[string]string `json:"header"`
}

func main() {
	logger.Printf("Listening on %s", Config.Address)
	log.Fatal(http.ListenAndServe(Config.Address, newServer()))
}

func newServer() http.Handler {
	router := mux.NewRouter()

	o := router.PathPrefix("/{user}/{repo}/objects").Subrouter()
	o.Methods("POST").Headers("Accept", metaMediaType).HandlerFunc(PostHandler)

	s := o.Path("/{oid}").Subrouter()
	s.Methods("GET", "HEAD").Headers("Accept", contentMediaType).HandlerFunc(GetContentHandler)
	s.Methods("GET", "HEAD").Headers("Accept", metaMediaType).HandlerFunc(GetMetaHandler)
	s.Methods("OPTIONS").Headers("Accept", contentMediaType).HandlerFunc(OptionsHandler)
	s.Methods("PUT").Headers("Accept", contentMediaType).HandlerFunc(PutHandler)

	return router
}

func GetContentHandler(w http.ResponseWriter, r *http.Request) {
	meta, err := getMeta(r)
	if err != nil {
		w.WriteHeader(404)
		logRequest(r, 404)
		return
	}

	token := S3NewToken("GET", oidPath(meta.Oid), meta.Oid)
	header := w.Header()
	header.Set("Git-Media-Set-Date", token.Time.Format(http.TimeFormat))
	header.Set("Git-Media-Set-Authorization", token.Token)
	header.Set("Git-Media-Set-x-amz-content-sha256", meta.Oid)
	header.Set("Location", token.Location)
	w.WriteHeader(302)
	logRequest(r, 302)
}

func GetMetaHandler(w http.ResponseWriter, r *http.Request) {
	m, err := getMeta(r)
	if err != nil {
		w.WriteHeader(404)
		fmt.Fprint(w, `{"message":"Not Found"}`)
		logRequest(r, 404)
		return
	}

	w.Header().Set("Content-Type", metaMediaType)

	meta := newMeta(m, false)
	enc := json.NewEncoder(w)
	enc.Encode(meta)
	logRequest(r, 200)
}

func PostHandler(w http.ResponseWriter, r *http.Request) {
	m, err := sendMeta(r)
	if err != nil {
		w.WriteHeader(404)
		fmt.Fprint(w, `{"message":"Not Found"}`)
		logRequest(r, 404)
		return
	}

	if !m.Writeable {
		w.WriteHeader(403)
		return
	}

	w.Header().Set("Content-Type", metaMediaType)

	if !m.existing {
		w.WriteHeader(201)
	}

	meta := newMeta(m, true)
	enc := json.NewEncoder(w)
	enc.Encode(meta)
	logRequest(r, 201)
}

func OptionsHandler(w http.ResponseWriter, r *http.Request) {
	m, err := getMeta(r)
	if err != nil {
		w.WriteHeader(404)
		logRequest(r, 404)
		return
	}

	if !m.Writeable {
		w.WriteHeader(403)
		logRequest(r, 403)
		return
	}

	if m.Oid == "" {
		w.WriteHeader(204)
		logRequest(r, 204)
	}

	logRequest(r, 200)
}

func PutHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(405)
	logRequest(r, 405)
}

func getMeta(r *http.Request) (*apiMeta, error) {
	vars := mux.Vars(r)
	user := vars["user"]
	repo := vars["repo"]
	oid := vars["oid"]

	authz := r.Header.Get("Authorization")
	url := Config.MetaEndpoint + "/" + filepath.Join(user, repo, oid)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Printf("[META] error - %s", err)
		return nil, err
	}
	if authz != "" {
		req.Header.Set("Authorization", authz)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Printf("[META] error - %s", err)
		return nil, err
	}

	defer res.Body.Close()
	if res.StatusCode == 200 {
		var m apiMeta
		dec := json.NewDecoder(res.Body)
		err := dec.Decode(&m)
		if err != nil {
			logger.Printf("[META] error - %s", err)
			return nil, err
		}

		return &m, nil
	}

	logger.Printf("[META] status - %d", res.StatusCode)
	return nil, fmt.Errorf("status: %d", res.StatusCode)
}

func sendMeta(r *http.Request) (*apiMeta, error) {
	vars := mux.Vars(r)
	user := vars["user"]
	repo := vars["repo"]

	var m apiMeta
	dec := json.NewDecoder(r.Body)
	err := dec.Decode(&m)
	if err != nil {
		logger.Printf("[META] error - %s", err)
		return nil, err
	}

	authz := r.Header.Get("Authorization")
	url := Config.MetaEndpoint + "/" + filepath.Join(user, repo, m.Oid)

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.Encode(&m)

	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		logger.Printf("[META] error - %s", err)
		return nil, err
	}
	if authz != "" {
		req.Header.Set("Authorization", authz)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Printf("[META] error - %s", err)
		return nil, err
	}

	defer res.Body.Close()
	if res.StatusCode == 200 || res.StatusCode == 201 {
		var m apiMeta
		dec := json.NewDecoder(res.Body)
		err := dec.Decode(&m)
		if err != nil {
			logger.Printf("[META] error - %s", err)
			return nil, err
		}

		m.existing = res.StatusCode == 200

		return &m, nil
	}

	logger.Printf("[META] status - %d", res.StatusCode)
	return nil, fmt.Errorf("status: %d", res.StatusCode)
}

func newMeta(m *apiMeta, download bool) *Meta {
	meta := &Meta{
		Oid:   m.Oid,
		Size:  m.Size,
		Links: make(map[string]*link),
	}
	meta.Links["download"] = newLink("GET", meta.Oid)
	if download {
		meta.Links["upload"] = newLink("PUT", meta.Oid)
		meta.Links["callback"] = &link{Href: "http://example.com/callmemaybe"}
	}
	return meta
}

func newLink(method, oid string) *link {
	token := S3NewToken(method, oidPath(oid), oid)
	header := make(map[string]string)
	header["Date"] = token.Time.Format(http.TimeFormat)
	header["Authorization"] = token.Token
	header["x-amz-content-sha256"] = oid

	return &link{Href: token.Location, Header: header}
}

func oidPath(oid string) string {
	dir := filepath.Join(oid[0:2], oid[2:4])

	return filepath.Join("/", dir, oid)
}

func logRequest(r *http.Request, status int) {
	logger.Printf("[%s] %s - %d", r.Method, r.URL, status)
}
