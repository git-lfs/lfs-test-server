package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
)

type ContentStorer interface {
	Get(*Meta, http.ResponseWriter, *http.Request) int
	PutLink(*Meta) *link
	Verify(*Meta) (bool, error)
}

type MetaStorer interface {
	Get(*RequestVars) (*Meta, error)
	Send(*RequestVars) (*Meta, error)
	Verify(*RequestVars) error
}

type RequestVars struct {
	Oid           string
	Size          int64
	User          string
	Repo          string
	Authorization string
	PathPrefix    string
	Status        int64
	Body          string
}

type Meta struct {
	Oid        string `json:"oid"`
	Size       int64  `json:"size"`
	PathPrefix string `json:"path_prefix"`
	existing   bool
}

type Representation struct {
	Oid   string
	Size  int64
	Links map[string]*link `json:"_links"`
}

func (v *RequestVars) ObjectLink() string {
	path := fmt.Sprintf("/%s/%s/objects/%s", v.User, v.Repo, v.Oid)
	return fmt.Sprintf("%s://%s%s", Config.Scheme, Config.Host, path)
}

type link struct {
	Href   string            `json:"href"`
	Header map[string]string `json:"header,omitempty"`
}

var (
	apiAuthError = errors.New("auth error")
)

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
	s.Options(contentMediaType, app.OptionsHandler)
	s.Put(contentMediaType, app.PutHandler)
	s.Post(metaMediaType, app.CallbackHandler)

	o := r.Route("/{user}/{repo}/objects")
	o.Post(metaMediaType, app.PostHandler)

	app.Router = r

	return app
}

func (a *App) Serve(l net.Listener) error {
	return http.Serve(l, a.Router)
}

func (a *App) GetContentHandler(w http.ResponseWriter, r *http.Request) {
	rv := unpack(r)
	meta, err := a.MetaStore.Get(rv)
	if err != nil {
		w.WriteHeader(404)
		logRequest(r, 404)
		return
	}

	status := a.ContentStore.Get(meta, w, r)

	w.WriteHeader(status)
	logRequest(r, status)
}

func (a *App) GetMetaHandler(w http.ResponseWriter, r *http.Request) {
	rv := unpack(r)
	meta, err := a.MetaStore.Get(rv)
	if err != nil {
		w.WriteHeader(404)
		fmt.Fprint(w, `{"message":"Not Found"}`)
		logRequest(r, 404)
		return
	}

	w.Header().Set("Content-Type", metaMediaType)

	if r.Method == "GET" {
		enc := json.NewEncoder(w)
		enc.Encode(a.Represent(rv, meta, false))
	}

	logRequest(r, 200)
}

func (a *App) PostHandler(w http.ResponseWriter, r *http.Request) {
	rv := unpack(r)
	meta, err := a.MetaStore.Send(rv)

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

	sentStatus := 200
	if !meta.existing {
		sentStatus = 201
		w.WriteHeader(201)
	}

	enc := json.NewEncoder(w)
	enc.Encode(a.Represent(rv, meta, true))
	logRequest(r, sentStatus)
}

func (a *App) OptionsHandler(w http.ResponseWriter, r *http.Request) {
	av := unpack(r)
	m, err := a.MetaStore.Get(av)

	if err == apiAuthError {
		logRequest(r, 403)
		w.WriteHeader(403)
		return
	}

	if err != nil {
		w.WriteHeader(404)
		logRequest(r, 404)
		return
	}

	if m.Oid == "" {
		w.WriteHeader(204)
		logRequest(r, 204)
	}

	logRequest(r, 200)
}

func (a *App) PutHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", "GET, HEAD, POST, OPTIONS")
	w.WriteHeader(405)
	logRequest(r, 405)
}

func (a *App) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	rv := unpack(r)

	meta, err := a.MetaStore.Get(rv)
	if err != nil {
		w.WriteHeader(404)
		logRequest(r, 404)
		return
	}

	ok, err := a.ContentStore.Verify(meta)
	if !ok || err != nil {
		logRequest(r, 404) // Probably need to log an error
		w.WriteHeader(404)
		return
	}

	err = a.MetaStore.Verify(rv)

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
	fmt.Fprint(w, `{"message":"ok"}`)
	logRequest(r, 200)
}

func (a *App) Represent(rv *RequestVars, meta *Meta, upload bool) *Representation {
	rep := &Representation{
		Oid:   meta.Oid,
		Size:  meta.Size,
		Links: make(map[string]*link),
	}

	rep.Links["download"] = &link{Href: rv.ObjectLink()}
	if upload {
		rep.Links["upload"] = a.ContentStore.PutLink(meta)

		header := make(map[string]string)
		header["Accept"] = metaMediaType
		header["Authorization"] = rv.Authorization
		header["PathPrefix"] = meta.PathPrefix
		rep.Links["callback"] = &link{Href: rv.ObjectLink(), Header: header}
	}
	return rep
}

func unpack(r *http.Request) *RequestVars {
	vars := Vars(r)
	rv := &RequestVars{
		User:          vars["user"],
		Repo:          vars["repo"],
		Oid:           vars["oid"],
		Authorization: r.Header.Get("Authorization"),
		PathPrefix:    r.Header.Get("PathPrefix"),
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
	logger.Printf("[%s] %s - %d", r.Method, r.URL, status)
}
