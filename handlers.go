package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/shirou/gopsutil/mem"
)

// NameHandler persists a count with the name supplied in the Request
// The name supplied in the Request is output to the end user
func NameHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	name := vars["name"]

	CountName(name)

	fmt.Fprintf(writer, "Hello, %s!", name)
}

// DefaultHandler outputs a simple welcome message to the end user
func DefaultHandler(writer http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(writer, "Welcome to the Hello Service")
}

// CountsHandler retrieves the list of names persisted with their counts (see NameHandler)
// The resulting Names instance is output in JSON to the end user
func CountsHandler(writer http.ResponseWriter, request *http.Request) {
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
}

// HealthHandler retrieves a list of information about the node
// The resulting NodeHealth instance is output in JSON to the end user
func HealthHandler(writer http.ResponseWriter, request *http.Request) {
	// Loads up Memory stats
	vm, err := mem.VirtualMemory()
	if err != nil {
		fmt.Fprintf(writer, err.Error())
		panic(err.Error())
	}

	// Initializing nodeHealth dataset
	nodeHealth := NodeHealth{
		EC2InstanceID:          GetInstanceID(writer),
		UpTime:                 GetUptime(writer),
		CPUPercent:             GetCPUUtilization(writer),
		DiskPercent:            GetDiskUtilization(writer),
		RAMTotalBytesUsed:      vm.Used,
		RAMTotalBytesAvailable: vm.Available,
	}

	jsonResponse, _ := json.Marshal(nodeHealth)
	fmt.Fprintf(writer, "%s", string(jsonResponse))
}

// ClusterHealthHandler retrieves health information for nodes in the cluster
// The resulting ClusterHealth is output in JSON to the end user
func ClusterHealthHandler(writer http.ResponseWriter, request *http.Request) {
	clusterHealth := GetClusterHealth(writer)

	jsonResponse, _ := json.Marshal(clusterHealth)
	fmt.Fprintf(writer, "%s", jsonResponse)

}
