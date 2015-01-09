package main

import (
	"github.com/gorilla/mux"
	"log"
	"net/http"
)

func main() {
	router := mux.NewRouter()

	s := router.Path("/{user}/{repo}/objects/{oid}").Subrouter()

	s.Methods("PUT").HandlerFunc(PutHandler)
	s.Methods("GET").HandlerFunc(GetHandler)

	log.Fatal(http.ListenAndServe(":8083", router))
}

func PutHandler(w http.ResponseWriter, r *http.Request) {
}

func GetHandler(w http.ResponseWriter, r *http.Request) {
}
