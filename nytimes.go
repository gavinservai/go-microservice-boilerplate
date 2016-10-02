package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/garyburd/redigo/redis"
)

const (
	// RedisAddress is the address of the Redis cluster
	RedisAddress = "127.0.0.1:6379"
)

var (
	c, err = redis.Dial("tcp", RedisAddress)
)

// Names represents a set of Name
type Names struct {
	Counts []Name `json:"counts"`
}

// Name represents a tuple of a name, and its count (the number of times the name was called via the /hello endpoint)
type Name struct {
	Name  string `json:"name"`
	Count string `json:"count"`
}

// CountName takes the supplied string, and increments a score for it on a Redis Sorted Set
func CountName(name string) {
	c.Send("ZINCRBY", "nytimes.names", 1, name)
	c.Flush()
	fmt.Println(c.Receive())
}

func main() {
	// Handle the Request
	HandleRequest()

	log.Fatal(http.ListenAndServe(":8081", nil))
}
