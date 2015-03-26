package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
)

type ContentStorer interface {
	Get(*Meta) (io.Reader, error)

	Put(*Meta, io.Reader) error
}

// MetaStorer provides an interface for serving object metadata.
type MetaStorer interface {
	// Get fetches an object's metadata from the metadata storage service.
	Get(*RequestVars) (*Meta, error)

	// Send sends an object's metadata to the metadata storage service.
	Put(*RequestVars) (*Meta, error)
}

// RequestVars contain variables from the HTTP request. Variables from routing, json body decoding, and
// some headers are stored.
type RequestVars struct {
	Oid           string
	Size          int64
	User          string
	Password      string
	Repo          string
	Authorization string
	PathPrefix    string
	Status        int64
	Body          string
	RequestID     string
}

// Meta is object metadata as seen by the object and metadata stores.
type Meta struct {
	Oid        string `json:"oid"`
	Size       int64  `json:"size"`
	PathPrefix string `json:"path_prefix"`
	existing   bool
}

// Representation is object medata as seen by clients of harbour.
type Representation struct {
	Oid   string
	Size  int64
	Links map[string]*link `json:"_links"`
}

// ObjectLink builds a URL linking to the object.
func (v *RequestVars) ObjectLink() string {
	path := fmt.Sprintf("/%s/%s/objects/%s", v.User, v.Repo, v.Oid)
	return fmt.Sprintf("%s://%s%s", Config.Scheme, Config.Host, path)
}

// link provides a structure used to build a hypermedia representation of an HTTP link.
type link struct {
	Href   string            `json:"href"`
	Header map[string]string `json:"header,omitempty"`
}

type App struct {
	Router       *Router
	ContentStore ContentStorer
	MetaStore    MetaStorer
}

func NewApp(content ContentStorer, meta MetaStorer) *App {
	app := &App{ContentStore: content, MetaStore: meta}

	r := NewRouter()

	s := r.Route("/{user}/{repo}/objects/{oid}")
	s.Get(contentMediaType, app.GetContentHandler)
	s.Head(contentMediaType, app.GetContentHandler)
	s.Get(metaMediaType, app.GetMetaHandler)
	s.Head(metaMediaType, app.GetMetaHandler)
	s.Put(contentMediaType, app.PutHandler)

	o := r.Route("/{user}/{repo}/objects")
	o.Post(metaMediaType, app.PostHandler)

	app.Router = r

	return app
}

func (a *App) Serve(l net.Listener) error {
	return http.Serve(l, a.Router)
}

// Download the data
func (a *App) GetContentHandler(w http.ResponseWriter, r *http.Request) {
	rv := unpack(r)
	meta, err := a.MetaStore.Get(rv)
	if err != nil {
		w.WriteHeader(404)
		logRequest(r, 404)
		return
	}

	content, err := a.ContentStore.Get(meta)
	if err != nil {
		w.WriteHeader(404)
		logRequest(r, 404)
		return
	}

	io.Copy(w, content)
	logRequest(r, 200)
}

// Just get the metadata
func (a *App) GetMetaHandler(w http.ResponseWriter, r *http.Request) {
	rv := unpack(r)
	meta, err := a.MetaStore.Get(rv)
	if err != nil {
		if isAuthError(err) {
			w.WriteHeader(401)
			fmt.Fprintf(w, `{"message":"Forbidden"}`)
			logRequest(r, 401)
		} else {
			w.WriteHeader(404)
			fmt.Fprint(w, `{"message":"Not Found"}`)
			logRequest(r, 404)
		}
		return
	}

	w.Header().Set("Content-Type", metaMediaType)

	if r.Method == "GET" {
		enc := json.NewEncoder(w)
		enc.Encode(a.Represent(rv, meta, false))
	}

	logRequest(r, 200)
}

// Request to upload
func (a *App) PostHandler(w http.ResponseWriter, r *http.Request) {
	rv := unpack(r)
	meta, err := a.MetaStore.Put(rv)
	if err != nil {
		if isAuthError(err) {
			w.WriteHeader(401)
			fmt.Fprint(w, `{"message":"Forbidden"}`)
			logRequest(r, 401)
		} else {
			w.WriteHeader(404)
			fmt.Fprint(w, `{"message":"Not Found"}`)
			logRequest(r, 404)
		}
		return
	}

	w.Header().Set("Content-Type", metaMediaType)

	sentStatus := 200
	if !meta.existing {
		sentStatus = 201
		w.WriteHeader(201)
	}

	enc := json.NewEncoder(w)
	enc.Encode(a.Represent(rv, meta, true))
	logRequest(r, sentStatus)
}

// Handle the upload
func (a *App) PutHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", "GET, HEAD, POST, OPTIONS")
	w.WriteHeader(405)
	logRequest(r, 405)
}

func (a *App) Represent(rv *RequestVars, meta *Meta, upload bool) *Representation {
	rep := &Representation{
		Oid:   meta.Oid,
		Size:  meta.Size,
		Links: make(map[string]*link),
	}

	rep.Links["download"] = &link{Href: rv.ObjectLink()}
	if upload {
		header := make(map[string]string)
		header["Accept"] = metaMediaType
		header["Authorization"] = rv.Authorization
		rep.Links["upload"] = &link{Href: rv.ObjectLink(), Header: header}
	}
	return rep
}

func unpack(r *http.Request) *RequestVars {
	vars := Vars(r)
	user, pass, _ := r.BasicAuth()

	rv := &RequestVars{
		User:          user,
		Password:      pass,
		Repo:          vars["repo"],
		Oid:           vars["oid"],
		Authorization: r.Header.Get("Authorization"),
		PathPrefix:    r.Header.Get("PathPrefix"),
		RequestID:     vars["request_id"],
	}

	if r.Method == "POST" { // Maybe also check if +json
		var p RequestVars
		dec := json.NewDecoder(r.Body)
		err := dec.Decode(&p)
		if err != nil {
			return rv
		}

		rv.Oid = p.Oid
		rv.Size = p.Size
		rv.Status = p.Status
		rv.Body = p.Body
	}

	return rv
}

func logRequest(r *http.Request, status int) {
	logger.Log(D{"method": r.Method, "url": r.URL, "status": status, "request_id": Vars(r)["request_id"]})
}

func isAuthError(err error) bool {
	type autherror interface {
		AuthError() bool
	}
	if ae, ok := err.(autherror); ok {
		return ae.AuthError()
	}
	return false
}
