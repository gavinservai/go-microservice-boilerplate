package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/garyburd/redigo/redis"
)

var (
	redisAddress = os.Getenv("REDIS_CLUSTER_ADDRESS")
	c, err       = redis.Dial("tcp", redisAddress)
)

// Names represents a set of Name (see Name struct)
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

	log.Fatal(http.ListenAndServe(":80", nil))
}
