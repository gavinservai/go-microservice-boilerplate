package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/disk"
	"github.com/shirou/gopsutil/host"
)

// NodeHealth represents a dataset of basic health information for a node
type NodeHealth struct {
	EC2InstanceID          string  `json:"ec2_instance_id"`
	UpTime                 uint64  `json:"uptime"`
	CPUPercent             float64 `json:"cpu_utilization_percent"`
	DiskPercent            float64 `json:"disk_utilization_percent"`
	RAMTotalBytesUsed      uint64  `json:"total_ram_bytes_used"`
	RAMTotalBytesAvailable uint64  `json:"total_ram_bytes_available"`
}

// ClusterHealth represents a set of NodeHealth
type ClusterHealth struct {
	NodeHealths []NodeHealth `json:"node_healths"`
}

// GetInstanceID retrieves the ec2 instance id
// If in DEBUG mode, "local" is returned
func GetInstanceID(writer http.ResponseWriter) string {
	ec2InstanceID := "local"
	if os.Getenv("DEBUG") != "true" {
		idResp, err := http.Get("http://169.254.169.254/latest/meta-data/instance-id")
		if err != nil {
			fmt.Fprintf(writer, err.Error())
			panic(err.Error())
		}
		defer idResp.Body.Close()

		ec2Id, err := ioutil.ReadAll(idResp.Body)
		if err != nil {
			fmt.Fprintf(writer, err.Error())
			panic(err.Error())
		}

		ec2InstanceID = string(ec2Id[:])
	}

	return ec2InstanceID
}

// GetUptime retrieves the Host Uptime
func GetUptime(writer http.ResponseWriter) uint64 {
	upTime, err := host.Uptime()
	if err != nil {
		fmt.Fprintf(writer, err.Error())
		panic(err.Error())
	}

	return upTime
}

// GetCPUUtilization retrieves the CPU Utilization percent
func GetCPUUtilization(writer http.ResponseWriter) float64 {
	cpuPercent, err := cpu.Percent(0, false)
	if err != nil {
		fmt.Fprintf(writer, err.Error())
		panic(err.Error())
	}

	return cpuPercent[0]
}

// GetDiskUtilization retrieves the Disk Utilization percent. This involves summing the utilization of the partitions
func GetDiskUtilization(writer http.ResponseWriter) float64 {
	// Get disk partitions
	diskPartitions, err := disk.Partitions(false)
	if err != nil {
		fmt.Fprintf(writer, err.Error())
		panic(err.Error())
	}

	// Sum up utilization of the disk partitions
	diskUtilization := 0.0
	for _, partition := range diskPartitions {
		u, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			fmt.Fprintf(writer, err.Error())
			panic(err.Error())
		}

		diskUtilization += u.UsedPercent
	}

	return diskUtilization
}

// GetClusterHealth retrieves the health of the nodes in the cluster
func GetClusterHealth(writer http.ResponseWriter) *ClusterHealth {
	var ipAddresses []string
	if os.Getenv("DEBUG") == "true" {
		ipAddresses = append(ipAddresses, "localhost")
	} else {
		sess := getSession(writer)
		ec2ids := getEC2IdsFromELB(writer, sess)
		ipAddresses = getIPAddressesFromEC2Ids(writer, sess, ec2ids)
	}

	clusterHealth := queryNodeHealths(writer, ipAddresses)

	return clusterHealth
}

// getSession instantiates a Session on AWS
// This method is only invoked on the deployed environment
func getSession(writer http.ResponseWriter) *session.Session {
	// Create new session
	sess := session.New(&aws.Config{Region: aws.String("us-west-2")})
	if err != nil {
		fmt.Fprintf(writer, err.Error())
		panic(err.Error())
	}

	return sess
}

// getEC2IdsFromELB retrieves EC2 Instance Ids from the ELB
// This method is only invoked on the deployed environment
func getEC2IdsFromELB(writer http.ResponseWriter, sess *session.Session) []*string {
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
		panic(err.Error())
	}

	var ec2ids []*string
	for _, elbInstance := range elbResponse.LoadBalancerDescriptions[0].Instances {
		ec2ids = append(ec2ids, elbInstance.InstanceId)
	}

	return ec2ids
}

// getIPAddressesFromEC2Ids retrieves IP Addresses from EC2 Instance Ids
// This method is only invoked on the deployed environment
func getIPAddressesFromEC2Ids(writer http.ResponseWriter, sess *session.Session, ec2ids []*string) []string {
	// Instantiate the EC2 Service, and use it to retrieve server IP Addresses from instance id's
	ec2Service := ec2.New(sess)
	describeInstancesParams := &ec2.DescribeInstancesInput{
		InstanceIds: ec2ids,
	}

	ec2Instances, err := ec2Service.DescribeInstances(describeInstancesParams)
	if err != nil {
		fmt.Fprintf(writer, err.Error())
		panic(err.Error())
	}

	var instanceIPAddresses []string
	for _, ec2Reservation := range ec2Instances.Reservations {
		for _, ec2Instance := range ec2Reservation.Instances {
			instanceIPAddresses = append(instanceIPAddresses, *ec2Instance.PrivateIpAddress)
		}
	}

	return instanceIPAddresses
}

// queryNodeHealths invokes the health endpoint on each node, and returns a ClusterHealth instance
func queryNodeHealths(writer http.ResponseWriter, instanceIPAddresses []string) *ClusterHealth {
	// Invoke the health endpoint on each node, and store the results
	clusterHealth := new(ClusterHealth)
	for _, IPAddress := range instanceIPAddresses {
		queryURL := "http://" + IPAddress + "/health"
		nodeHealthResp, err := http.Get(queryURL)
		if err != nil {
			fmt.Fprintf(writer, err.Error())
			panic(err.Error())
		}
		defer nodeHealthResp.Body.Close()

		nodeHealth, err := ioutil.ReadAll(nodeHealthResp.Body)
		if err != nil {
			fmt.Fprintf(writer, err.Error())
			panic(err.Error())
		}

		var nodeHealthData NodeHealth
		json.Unmarshal(nodeHealth, &nodeHealthData)
		clusterHealth.NodeHealths = append(clusterHealth.NodeHealths, nodeHealthData)
	}

	return clusterHealth
}
