package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
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
	fmt.Fprintf(writer, "Welcome to the hello service")
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
}

// ClusterHealthHandler retrieves health information for nodes in the cluster
// Note: Only works on deployed AWS environment
func ClusterHealthHandler(writer http.ResponseWriter, request *http.Request) {
	// Create new session
	sess := session.New(&aws.Config{Region: aws.String("us-west-2")})
	if err != nil {
		fmt.Fprintf(writer, "Failed to create session")
	}

	// Retrieve list of EC2 Instance Id's of nodes currently on the load balancer
	elbService := elb.New(sess)
	elbParams := &elb.DescribeLoadBalancersInput{
		LoadBalancerNames: []*string{
			aws.String("hello-service-elb"),
		},
	}
	elbResponse, err := elbService.DescribeLoadBalancers(elbParams)

	if err != nil {
		fmt.Fprintf(writer, err.Error())
		return
	}

	var ec2ids []*string
	for _, elbInstance := range elbResponse.LoadBalancerDescriptions[0].Instances {
		ec2ids = append(ec2ids, elbInstance.InstanceId)
	}

	// Instantiate the EC2 Service, and use it to retrieve server IP Addresses from instance id's
	ec2Service := ec2.New(sess)
	describeInstancesParams := &ec2.DescribeInstancesInput{
		InstanceIds: ec2ids,
	}
	ec2Instances, err := ec2Service.DescribeInstances(describeInstancesParams)

	if err != nil {
		fmt.Fprintf(writer, err.Error())
		return
	}

	var instanceIPAddresses []string
	for _, ec2Reservation := range ec2Instances.Reservations {
		for _, ec2Instance := range ec2Reservation.Instances {
			instanceIPAddresses = append(instanceIPAddresses, *ec2Instance.PrivateIpAddress)
		}
	}

	// Invoke the health endpoint on each node, and store the results
	clusterHealth := new(ClusterHealth)
	for _, IPAddress := range instanceIPAddresses {
		queryURL := "http://" + IPAddress + "/health"
		nodeHealthResp, err := http.Get(queryURL)
		if err != nil {
			fmt.Fprintf(writer, err.Error())
			nodeHealthResp.Body.Close()
			return
		}

		nodeHealth, err := ioutil.ReadAll(nodeHealthResp.Body)
		if err != nil {
			fmt.Fprintf(writer, err.Error())
			nodeHealthResp.Body.Close()
			return
		}

		var nodeHealthData NodeHealth
		json.Unmarshal(nodeHealth, &nodeHealthData)
		clusterHealth.NodeHealths = append(clusterHealth.NodeHealths, nodeHealthData)
		nodeHealthResp.Body.Close()
	}

	jsonResponse, _ := json.Marshal(clusterHealth)
	fmt.Fprintf(writer, "%s", jsonResponse)

}
