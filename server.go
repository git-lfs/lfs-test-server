package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
)

type Meta struct {
	Oid   string           `json:"oid"`
	Size  int64            `json:"size"`
	Links map[string]*link `json:"_links,omitempty"`
}

type apiMeta struct {
	Oid        string `json:"oid"`
	Size       int64  `json:"size"`
	Writeable  bool   `json:"writeable"`
	PathPrefix string `json:"path_prefix"`
	existing   bool   `json:"-"`
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
	apiAuthError = errors.New("auth error")
)

func newRouter() http.Handler {
	r := NewRouter()

	s := r.Route("/{user}/{repo}/objects/{oid}")
	s.Get(contentMediaType, GetContentHandler)
	s.Head(contentMediaType, GetContentHandler)
	s.Get(metaMediaType, GetMetaHandler)
	s.Head(metaMediaType, GetMetaHandler)
	s.Options(contentMediaType, OptionsHandler)
	s.Put(contentMediaType, PutHandler)

	o := r.Route("/{user}/{repo}/objects")
	o.Post(metaMediaType, PostHandler)

	return r
}

func GetContentHandler(w http.ResponseWriter, r *http.Request) {
	av := unpack(r)
	meta, err := GetMeta(av)
	if err != nil {
		w.WriteHeader(404)
		logRequest(r, 404)
		return
	}

	token := S3SignQuery("GET", path.Join("/", meta.PathPrefix, oidPath(meta.Oid)), 86400)
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

	if err == apiAuthError {
		logRequest(r, 403)
		w.WriteHeader(403)
		return
	}

	if err != nil {
		w.WriteHeader(404)
		fmt.Fprint(w, `{"message":"Not Found"}`)
		logRequest(r, 404)
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
	url := Config.MetaEndpoint + "/" + path.Join("internal/repos", v.User, v.Repo, "media", "blobs", v.Oid)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Printf("[META] error - %s", err)
		return nil, err
	}
	req.Header.Set("Accept", Config.ApiMediaType)
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

		logger.Printf("[META] status - %d", res.StatusCode)
		return &m, nil
	}

	logger.Printf("[META] status - %d", res.StatusCode)
	return nil, fmt.Errorf("status: %d", res.StatusCode)
}

func SendMeta(v *appVars) (*apiMeta, error) {
	url := Config.MetaEndpoint + "/" + path.Join("internal/repos", v.User, v.Repo, "media", "blobs", v.Oid)

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.Encode(&apiMeta{Oid: v.Oid, Size: v.Size})

	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		logger.Printf("[META] error - %s", err)
		return nil, err
	}
	req.Header.Set("Accept", Config.ApiMediaType)
	if v.Authorization != "" {
		req.Header.Set("Authorization", v.Authorization)
	}

	if Config.HmacKey != "" {
		mac := hmac.New(sha256.New, []byte(Config.HmacKey))
		mac.Write(buf.Bytes())
		req.Header.Set("Content-Hmac", "sha256 "+hex.EncodeToString(mac.Sum(nil)))
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Printf("[META] error - %s", err)
		return nil, err
	}

	defer res.Body.Close()

	if res.StatusCode == 403 {
		return nil, apiAuthError
	}

	if res.StatusCode == 200 || res.StatusCode == 201 {
		var m apiMeta
		dec := json.NewDecoder(res.Body)
		err := dec.Decode(&m)
		if err != nil {
			logger.Printf("[META] error - %s", err)
			return nil, err
		}
		m.Writeable = true // Probably remove this
		m.existing = res.StatusCode == 200
		logger.Printf("[META] status - %d", res.StatusCode)
		return &m, nil
	}

	logger.Printf("[META] status - %d", res.StatusCode)
	return nil, fmt.Errorf("status: %d", res.StatusCode)
}

func unpack(r *http.Request) *appVars {
	vars := Vars(r)
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

	path := fmt.Sprintf("/%s/%s/objects/%s", v.User, v.Repo, m.Oid)
	meta.Links["download"] = &link{Href: fmt.Sprintf("https://%s%s", Config.Host, path)}
	if upload {
		meta.Links["upload"] = signedLink("PUT", m.PathPrefix, meta.Oid)
		meta.Links["callback"] = &link{Href: "http://example.com/callmemaybe"}
	}
	return meta
}

func signedLink(method, pathPrefix, oid string) *link {
	token := S3SignHeader(method, path.Join("/", pathPrefix, oidPath(oid)), oid)
	header := make(map[string]string)
	header["Authorization"] = token.Token
	header["x-amz-content-sha256"] = oid
	header["x-amz-date"] = token.Time.Format(isoLayout)

	return &link{Href: token.Location, Header: header}
}

func oidPath(oid string) string {
	dir := path.Join(oid[0:2], oid[2:4])

	return path.Join("/", dir, oid)
}

func logRequest(r *http.Request, status int) {
	logger.Printf("[%s] %s - %d", r.Method, r.URL, status)
}
