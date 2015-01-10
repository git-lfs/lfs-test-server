package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"path/filepath"
)

func main() {
	router := mux.NewRouter()

	s := router.Path("/{user}/{repo}/objects/{oid}").Subrouter()

	s.Methods("GET").Headers("Accept", "application/vnd.git-media").HandlerFunc(GetHandler)
	s.Methods("GET").Headers("Accept", "application/vnd.git-media+json").HandlerFunc(GetUploadHandler)

	log.Fatal(http.ListenAndServe(":8083", router))
}

func GetHandler(w http.ResponseWriter, r *http.Request) {

}

func GetUploadHandler(w http.ResponseWriter, r *http.Request) {
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

	enc := json.NewEncoder(w)
	enc.Encode(m)
}

func oidPath(oid string) string {
	dir := filepath.Join(oid[0:2], oid[2:4])

	return filepath.Join("/", dir, oid)
}
