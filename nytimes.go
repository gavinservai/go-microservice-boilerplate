package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
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
	// Initialize Region if the environment is not currently set (this will occur on AWS)
	_, regionExists := os.LookupEnv("AWS_REGION")
	if !regionExists {
		os.Setenv("AWS_REGION", "us-west-2")

		// If this is an EC2 Instance, the EC2 Metadata is retrieved, and the Region extracted from the JSON
		if os.Getenv("DEBUG") != "true" {
			metaDocumentResp, err := http.Get("http://169.254.169.254/latest/dynamic/instance-identity/document")
			if err != nil {
				panic(err.Error())
			}
			defer metaDocumentResp.Body.Close()

			metaDocumentString, err := ioutil.ReadAll(metaDocumentResp.Body)
			if err != nil {
				panic(err.Error())
			}

			var metaDocument ec2metadata.EC2InstanceIdentityDocument
			json.Unmarshal(metaDocumentString, &metaDocument)

			os.Setenv("AWS_REGION", metaDocument.Region)
		}
	}

	// Handle the Request
	HandleRequest()

	log.Fatal(http.ListenAndServe(":80", nil))
}
