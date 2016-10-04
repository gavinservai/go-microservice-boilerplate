package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
)

// NameHandler persists a count with the name supplied in the Request
// The name supplied in the Request is output to the end user
func NameHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	name := vars["name"]

	CountName(name)

	fmt.Fprintf(writer, "Hello, %q", name)
}

// Outputs a simple welcome message to the end user
func DefaultHandler(writer http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(writer, "Welcome")
}

// CountHandler retrieves the list of names persisted with their counts (see NameHandler)
// This list is output in JSON to the end user
func CountHandler(writer http.ResponseWriter, request *http.Request) {
	namesAndScores, _ := redis.Strings(c.Do("ZRANGE", "nytimes.names", 0, -1, "WITHSCORES"))

	counts := new(Names)
	countHolder := new(Name)
	for i, value := range namesAndScores {
		if (i+1)%2 == 0 {
			countHolder.Count = value
			counts.Counts = append(counts.Counts, *countHolder)
			countHolder = new(Name)
		} else {
			countHolder.Name = value
		}
	}

	jsonResponse, _ := json.Marshal(counts)
	fmt.Fprintf(writer, "%s", string(jsonResponse))
	fmt.Println(string(jsonResponse))
}
