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
	Oid       string           `json:"oid"`
	Size      int64            `json:"size"`
	Links     map[string]*link `json:"_links"`
	writeable bool             `json:"-"`
}

type link struct {
	Href   string            `json:"href"`
	Header map[string]string `json:"header"`
}

func main() {
	log.Fatal(http.ListenAndServe(":8083", newServer()))
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

// 200 - Serve the content
// 302 - Redirect to other content storage
// 404 - No access or content does not exist
func GetContentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := vars["user"]
	repo := vars["repo"]
	oid := vars["oid"]

	_, err := getMeta(user, repo, oid) // TODO - really needs to check auth
	if err != nil {
		log.Printf("getMeta error: %s", err)
		w.WriteHeader(404)
		return
	}

	token := S3NewToken("GET", oidPath(oid), oid)
	header := w.Header()
	header.Set("Git-Media-Set-Date", token.Time.Format(http.TimeFormat))
	header.Set("Git-Media-Set-Authorization", token.Token)
	header.Set("Git-Media-Set-x-amz-content-sha256", oid)
	header.Set("Location", token.Location)
	w.WriteHeader(302)
}

// 200 - Serve the metadata
// 403 - can read but not write
// 404 - can't access / repo does not exist for this user
func GetMetaHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	user := vars["user"]
	repo := vars["repo"]
	oid := vars["oid"]

	meta, err := getMeta(user, repo, oid)
	if err != nil {
		log.Printf("getMeta error: %s", err)
		w.WriteHeader(404)
		return
	}

	// Download link
	meta.Links["download"] = newLink("GET", oid)

	// Upload link, if it's writeable
	if meta.writeable {
		meta.Links["upload"] = newLink("PUT", oid)
		meta.Links["callback"] = &link{Href: "http://example.com/callmemaybe"}
	}

	w.Header().Set("Content-Type", metaMediaType)

	enc := json.NewEncoder(w)
	enc.Encode(meta)
}

// 200 - able to send, server has
// 204 - able to send, server does not have
// 403 - user can read but not write
// 404 - repo does not exist / no access
func OptionsHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(501)
}

// 200 - object already exists
// 201 - object uploaded successfully
// 409 - object contents do not match oid
// 403 - user can read but not write
// 404 - repo does not exist / no access
func PutHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(501)
}

func oidPath(oid string) string {
	dir := filepath.Join(oid[0:2], oid[2:4])

	return filepath.Join("/", dir, oid)
}

// getMeta validate's a user's access to the repo and gets object metadata
func getMeta(user, repo, oid string) (*Meta, error) {
	url := Config.MetaEndpoint + "/" + filepath.Join(user, repo, oid)
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == 200 || res.StatusCode == 404 {
		var meta Meta
		dec := json.NewDecoder(res.Body)
		err := dec.Decode(&meta)
		if err != nil {
		}
		meta.Links = make(map[string]*link)
		meta.writeable = true

		return &meta, nil
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
