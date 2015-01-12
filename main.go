package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"path/filepath"
)

const (
	contentMediaType = "application/vnd.git-media"
	metaMediaType    = contentMediaType + ".json"
)

func main() {
	router := mux.NewRouter()

	s := router.Path("/{user}/{repo}/objects/{oid}").Subrouter()

	s.Methods("GET", "HEAD").Headers("Accept", contentMediaType).HandlerFunc(GetContentHandler)
	s.Methods("GET", "HEAD").Headers("Accept", metaMediaType).HandlerFunc(GetMetaHandler)

	log.Fatal(http.ListenAndServe(":8083", router))
}

func GetContentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	oid := vars["oid"]
	path := oidPath(oid)
	token := S3NewToken("GET", path, oid)

	header := w.Header()
	header.Set("Git-Media-Set-Date", token.Time.Format(http.TimeFormat))
	header.Set("Git-Media-Set-Authorization", token.Token)
	header.Set("Git-Media-Set-x-amz-content-sha256", oid)
	header.Set("Location", token.Location)
	w.WriteHeader(302)
}

// Get the rest of the metadata
func GetMetaHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	oid := vars["oid"]
	path := oidPath(oid)
	token := S3NewToken("PUT", path, oid)

	m := make(map[string]map[string]string)
	header := make(map[string]string)
	header["Date"] = token.Time.Format(http.TimeFormat)
	header["Authorization"] = token.Token
	header["x-amz-content-sha256"] = oid
	m["header"] = header

	links := make(map[string]string)
	links["upload"] = token.Location
	links["callback"] = "http://somecallback.com"
	m["_links"] = links

	w.Header().Set("Content-Type", metaMediaType)
	enc := json.NewEncoder(w)
	enc.Encode(m)
}

func oidPath(oid string) string {
	dir := filepath.Join(oid[0:2], oid[2:4])

	return filepath.Join("/", dir, oid)
}
