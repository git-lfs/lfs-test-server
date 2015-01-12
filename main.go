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
	metaMediaType    = contentMediaType + ".json"
)

type Meta struct {
	Oid    string            `json:"oid"`
	Size   int64             `json:"size"`
	Header map[string]string `json:"header"`
	Links  map[string]string `json:"_links"`
	Exists bool              `json:"exists"`
}

func main() {
	router := mux.NewRouter()

	s := router.Path("/{user}/{repo}/objects/{oid}").Subrouter()

	s.Methods("GET", "HEAD").Headers("Accept", contentMediaType).HandlerFunc(GetContentHandler)
	s.Methods("GET", "HEAD").Headers("Accept", metaMediaType).HandlerFunc(GetMetaHandler)

	log.Fatal(http.ListenAndServe(":8083", router))
}

// 200 - Serve the content
// 302 - Redirect to other content storage
// 404 - No access or content does not exist
func GetContentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	oid := vars["oid"]

	_, err := getMeta(oid) // TODO - really needs to check auth
	if err != nil {
		log.Printf("getMeta error: %s", err)
		w.WriteHeader(404)
		return
	}

	token := S3NewToken("HEAD", oidPath(oid), oid)

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
	oid := vars["oid"]

	meta, err := getMeta(oid)
	if err != nil {
		log.Printf("getMeta error: %s", err)
		w.WriteHeader(404) // TODO real error
		return
	}

	token := S3NewToken("PUT", oidPath(oid), oid)

	meta.Header["Date"] = token.Time.Format(http.TimeFormat)
	meta.Header["Authorization"] = token.Token
	meta.Header["x-amz-content-sha256"] = oid

	meta.Links["upload"] = token.Location
	meta.Links["callback"] = "http://somecallback.com"

	w.Header().Set("Content-Type", metaMediaType)

	enc := json.NewEncoder(w)
	enc.Encode(meta)
}

func oidPath(oid string) string {
	dir := filepath.Join(oid[0:2], oid[2:4])

	return filepath.Join("/", dir, oid)
}

func getMeta(oid string) (*Meta, error) {
	url := fmt.Sprintf("https://%s.s3.amazonaws.com%s", Config.AwsBucket, oidPath(oid))
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}

	token := S3NewToken("HEAD", oidPath(oid), oid)
	req.Header.Add("date", token.Time.Format(http.TimeFormat))
	req.Header.Add("authorization", token.Token)
	req.Header.Add("x-amz-content-sha256", oid)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	switch res.StatusCode {
	case 200:
		return newMeta(oid, res.ContentLength, true), nil
	case 404:
		return newMeta(oid, 0, false), nil
	default:
		return nil, fmt.Errorf("s3 status: %d", res.StatusCode)
	}
}

func newMeta(oid string, size int64, exists bool) *Meta {
	return &Meta{
		Oid:    oid,
		Size:   size,
		Header: make(map[string]string),
		Links:  make(map[string]string),
		Exists: exists,
	}
}
