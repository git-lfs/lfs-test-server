package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"path/filepath"
)

const (
	contentMediaType = "application/vnd.git-media"
	metaMediaType    = contentMediaType + "+json"
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
}

type link struct {
	Href   string            `json:"href"`
	Header map[string]string `json:"header"`
}

func main() {
	log.Fatal(http.ListenAndServe(Config.Address, newServer()))
}

func newServer() http.Handler {
	router := mux.NewRouter()

	s := router.Path("/{user}/{repo}/objects/{oid}").Subrouter()

	s.Methods("GET", "HEAD").Headers("Accept", contentMediaType).HandlerFunc(GetContentHandler)
	s.Methods("GET", "HEAD").Headers("Accept", metaMediaType).HandlerFunc(GetMetaHandler)
	s.Methods("OPTIONS").Headers("Accept", contentMediaType).HandlerFunc(OptionsHandler)
	s.Methods("PUT").Headers("Accept", contentMediaType).HandlerFunc(PutHandler)

	return router
}

func GetContentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := vars["user"]
	repo := vars["repo"]
	oid := vars["oid"]

	meta, err := getMeta(r, user, repo, oid)
	if err != nil {
		w.WriteHeader(404)
		return
	}

	token := S3NewToken("GET", oidPath(meta.Oid), meta.Oid)
	header := w.Header()
	header.Set("Git-Media-Set-Date", token.Time.Format(http.TimeFormat))
	header.Set("Git-Media-Set-Authorization", token.Token)
	header.Set("Git-Media-Set-x-amz-content-sha256", meta.Oid)
	header.Set("Location", token.Location)
	w.WriteHeader(302)
}

func GetMetaHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := vars["user"]
	repo := vars["repo"]
	oid := vars["oid"]

	m, err := getMeta(r, user, repo, oid)
	if err != nil {
		w.WriteHeader(404)
		fmt.Fprint(w, `{"message":"Not Found"}`)
		return
	}

	meta := &Meta{
		Oid:   m.Oid,
		Size:  m.Size,
		Links: make(map[string]*link),
	}
	meta.Links["download"] = newLink("GET", oid)

	if m.Writeable {
		meta.Links["upload"] = newLink("PUT", oid)
		meta.Links["callback"] = &link{Href: "http://example.com/callmemaybe"}
	}

	w.Header().Set("Content-Type", metaMediaType)

	enc := json.NewEncoder(w)
	enc.Encode(meta)
}

func OptionsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := vars["user"]
	repo := vars["repo"]
	oid := vars["oid"]

	m, err := getMeta(r, user, repo, oid)
	if err != nil {
		w.WriteHeader(404)
		return
	}

	if !m.Writeable {
		w.WriteHeader(403)
		return
	}

	if m.Oid == "" {
		w.WriteHeader(204)
	}
}

func PutHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(405)
}

func oidPath(oid string) string {
	dir := filepath.Join(oid[0:2], oid[2:4])

	return filepath.Join("/", dir, oid)
}

// getMeta validate's a user's access to the repo and gets object metadata
func getMeta(r *http.Request, user, repo, oid string) (*apiMeta, error) {
	authz := r.Header.Get("Authorization")
	url := Config.MetaEndpoint + "/" + filepath.Join(user, repo, oid)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	if authz != "" {
		req.Header.Set("Authorization", authz)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == 200 {
		var m apiMeta
		dec := json.NewDecoder(res.Body)
		err := dec.Decode(&m)
		if err != nil {
			return nil, err
		}

		return &m, nil
	}

	return nil, fmt.Errorf("s3 status: %d", res.StatusCode)
}

func newLink(method, oid string) *link {
	token := S3NewToken(method, oidPath(oid), oid)
	header := make(map[string]string)
	header["Date"] = token.Time.Format(http.TimeFormat)
	header["Authorization"] = token.Token
	header["x-amz-content-sha256"] = oid

	return &link{Href: token.Location, Header: header}
}
