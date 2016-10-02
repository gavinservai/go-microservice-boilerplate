package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

// HandleRequest routes an HTTP request to its appropriate handler
func HandleRequest() {
	r := mux.NewRouter()
	r.HandleFunc("/hello/{name}", NameHandler)
	r.HandleFunc("/count", CountHandler)

	http.Handle("/", r)
}
