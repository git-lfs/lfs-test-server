package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"path/filepath"
)

type Meta struct {
	Oid   string           `json:"oid"`
	Size  int64            `json:"size"`
	Links map[string]*link `json:"_links,omitempty"`
}

type apiMeta struct {
	Oid       string `json:"oid"`
	Size      int64  `json:"size"`
	Writeable bool   `json:"writeable"`
	existing  bool   `json:"-"`
}

type link struct {
	Href   string            `json:"href"`
	Header map[string]string `json:"header,omitempty"`
}

type appVars struct {
	User          string
	Repo          string
	Oid           string
	Size          int64
	Authorization string
}

var (
	router *mux.Router
)

func newRouter() http.Handler {
	router = mux.NewRouter()

	o := router.PathPrefix("/{user}/{repo}/objects").Subrouter()
	o.Methods("POST").Headers("Accept", metaMediaType).HandlerFunc(PostHandler)
	o.Path("/{oid}").Methods("GET", "HEAD").Headers("Accept", contentMediaType).HandlerFunc(GetContentHandler).Name("download")
	o.Path("/{oid}").Methods("GET", "HEAD").Headers("Accept", metaMediaType).HandlerFunc(GetMetaHandler)
	o.Path("/{oid}").Methods("OPTIONS").Headers("Accept", contentMediaType).HandlerFunc(OptionsHandler)
	o.Path("/{oid}").Methods("PUT").Headers("Accept", contentMediaType).HandlerFunc(PutHandler)

	return router
}

func GetContentHandler(w http.ResponseWriter, r *http.Request) {
	av := unpack(r)
	meta, err := GetMeta(av)
	if err != nil {
		w.WriteHeader(404)
		logRequest(r, 404)
		return
	}

	token := S3SignQuery("GET", oidPath(meta.Oid), 86400)
	w.Header().Set("Location", token.Location)
	w.WriteHeader(302)
	logRequest(r, 302)
}

func GetMetaHandler(w http.ResponseWriter, r *http.Request) {
	av := unpack(r)
	m, err := GetMeta(av)
	if err != nil {
		w.WriteHeader(404)
		fmt.Fprint(w, `{"message":"Not Found"}`)
		logRequest(r, 404)
		return
	}

	w.Header().Set("Content-Type", metaMediaType)

	meta := newMeta(m, av, false)
	enc := json.NewEncoder(w)
	enc.Encode(meta)
	logRequest(r, 200)
}

func PostHandler(w http.ResponseWriter, r *http.Request) {
	av := unpack(r)
	m, err := SendMeta(av)
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

	meta := newMeta(m, av, true)
	enc := json.NewEncoder(w)
	enc.Encode(meta)
	logRequest(r, 201)
}

func OptionsHandler(w http.ResponseWriter, r *http.Request) {
	av := unpack(r)
	m, err := GetMeta(av)
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
	w.Header().Set("Allow", "GET, HEAD, POST, OPTIONS")
	w.WriteHeader(405)
	logRequest(r, 405)
}

func GetMeta(v *appVars) (*apiMeta, error) {
	url := Config.MetaEndpoint + "/" + filepath.Join(v.User, v.Repo, v.Oid)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Printf("[META] error - %s", err)
		return nil, err
	}
	if v.Authorization != "" {
		req.Header.Set("Authorization", v.Authorization)
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

func SendMeta(v *appVars) (*apiMeta, error) {
	url := Config.MetaEndpoint + "/" + filepath.Join(v.User, v.Repo, v.Oid)

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.Encode(&apiMeta{Oid: v.Oid, Size: v.Size})

	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		logger.Printf("[META] error - %s", err)
		return nil, err
	}
	if v.Authorization != "" {
		req.Header.Set("Authorization", v.Authorization)
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

func unpack(r *http.Request) *appVars {
	vars := mux.Vars(r)
	av := &appVars{
		User:          vars["user"],
		Repo:          vars["repo"],
		Oid:           vars["oid"],
		Authorization: r.Header.Get("Authorization"),
	}

	if r.Method == "POST" {
		var m appVars
		dec := json.NewDecoder(r.Body)
		err := dec.Decode(&m)
		if err != nil {
			return av
		}

		av.Oid = m.Oid
		av.Size = m.Size
	}

	return av
}

func newMeta(m *apiMeta, v *appVars, upload bool) *Meta {
	meta := &Meta{
		Oid:   m.Oid,
		Size:  m.Size,
		Links: make(map[string]*link),
	}

	path, _ := router.Get("download").URLPath("user", v.User, "repo", v.Repo, "oid", m.Oid)
	meta.Links["download"] = &link{Href: fmt.Sprintf("https://%s%s", Config.Host, path)}
	if upload {
		meta.Links["upload"] = signedLink("PUT", meta.Oid)
		meta.Links["callback"] = &link{Href: "http://example.com/callmemaybe"}
	}
	return meta
}

func signedLink(method, oid string) *link {
	token := S3SignHeader(method, oidPath(oid), oid)
	header := make(map[string]string)
	header["Authorization"] = token.Token
	header["x-amz-content-sha256"] = oid
	header["x-amz-date"] = token.Time.Format(isoLayout)

	return &link{Href: token.Location, Header: header}
}

func oidPath(oid string) string {
	dir := filepath.Join(oid[0:2], oid[2:4])

	return filepath.Join("/", dir, oid)
}

func logRequest(r *http.Request, status int) {
	logger.Printf("[%s] %s - %d", r.Method, r.URL, status)
}
