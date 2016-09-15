package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
)

const (
	REDIS_ADDRESS = "127.0.0.1:6379"
)

var (
	c, err = redis.Dial("tcp", REDIS_ADDRESS)
)

type Counts struct {
	Counts []Count `json:"counts"`
}

type Count struct {
	Name  string `json:"name"`
	Count string `json:"count"`
}

func NameHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	name := vars["name"]

	CountName(name)

	fmt.Fprintf(writer, "Hello, %q", name)
}

func CountName(name string) {
	c.Send("ZINCRBY", "nytimes.names", 1, name)
	c.Flush()
	fmt.Println(c.Receive())
}

func CountHandler(writer http.ResponseWriter, request *http.Request) {
	namesAndScores, _ := redis.Strings(c.Do("ZRANGE", "nytimes.names", 0, -1, "WITHSCORES"))
	// fmt.Printf("%v", names)

	counts := new(Counts)
	countHolder := new(Count)
	for i, value := range namesAndScores {
		if (i+1)%2 == 0 {
			countHolder.Count = value
			counts.Counts = append(counts.Counts, *countHolder)
			countHolder = new(Count)
		} else {
			countHolder.Name = value
		}
	}

	jsonResponse, _ := json.Marshal(counts)
	fmt.Println(string(jsonResponse))
}

func handleRequest() {
	r := mux.NewRouter()
	r.HandleFunc("/hello/{name}", NameHandler)
	r.HandleFunc("/count", CountHandler)

	http.Handle("/", r)
}

func main() {
	// Handle the Request
	handleRequest()

	log.Fatal(http.ListenAndServe(":8081", nil))
}
