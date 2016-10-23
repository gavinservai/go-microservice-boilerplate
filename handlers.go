package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/garyburd/redigo/redis"
	"github.com/gorilla/mux"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/mem"
)

// NameHandler persists a count with the name supplied in the Request
// The name supplied in the Request is output to the end user
func NameHandler(writer http.ResponseWriter, request *http.Request) {
	vars := mux.Vars(request)
	name := vars["name"]

	CountName(name)

	fmt.Fprintf(writer, "Hello, %q", name)
}

// DefaultHandler outputs a simple welcome message to the end user
func DefaultHandler(writer http.ResponseWriter, request *http.Request) {
	fmt.Fprintf(writer, "Welcome to the hello service 3")
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

// HealthHandler retrieves a list of information about the node
// This list is output in JSON to the end user
func HealthHandler(writer http.ResponseWriter, request *http.Request) {
	// Loads up Memory stats
	vm, _ := mem.VirtualMemory()

	// Retrieves the CPU Utilizatin percent
	cpuPercent, _ := cpu.Percent(0, false)

	// Retrieving the Disk Utilization percent. This involves summing the utilization of the partitions
	diskPartitions, _ := disk.Partitions(false)
	diskUtilization := 0.0

	for _, partition := range diskPartitions {
		u, _ := disk.Usage(partition.Mountpoint)
		diskUtilization += u.UsedPercent
	}

	// Retrieving the Host uptime
	upTime, _ := host.Uptime()

	// Initializing nodeHealth dataset
	nodeHealth := NodeHealth{
		UpTime:            upTime,
		CPUPercent:        cpuPercent[0],
		DiskPercent:       diskUtilization,
		RAMTotalUsed:      vm.Used,
		RAMTotalAvailable: vm.Available,
	}

	jsonResponse, _ := json.Marshal(nodeHealth)
	fmt.Fprintf(writer, "%s", string(jsonResponse))
	fmt.Println(string(jsonResponse))
}
