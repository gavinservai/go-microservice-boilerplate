Hello Service
=============

- [Overview](#overview)
  * [Infrastructure](#infrastructure)
- [Local Development](#local-development)
  * [Requirements](#requirements)
  * [Running the Service](#running-the-service)
  * [Stopping the Service](#stopping-the-service)
- [Infrastructure Deployment](#infrastructure-deployment)
  * [Requirements](#requirements-1)
  * [Deploying the Infrastructure](#deploying-the-infrastructure)
  * [Updating the Infrastructure](#updating-the-infrastructure)
  * [Destroying the Infrastructure](#destroying-the-infrastructure)
- [Release Deployment](#release-deployment)
  * [Requirements](#requirements-2)
  * [Updating the Service](#updating-the-service)
- [Endpoints](#endpoints)
  * [hello/{name}](#helloname)
  * [counts](#counts)
  * [health](#health)
  * [health/cluster](#healthcluster)
- [What Next?](#what-next)

----------

## Overview
Hello Service is a JSON micro-service written in [Go](https://golang.org/doc/) with a simple purpose - to view how many times the server has said hello to a name. (Refer to the [API](#endpoints) section for more details).

### Infrastructure
Hello Service makes use of various components and tools in order to simplify the process of management, maintenance, and deployment of the service. 

The following major AWS components are used:

- [EC2 Container Service (ECS)](https://aws.amazon.com/ecs/) - ECS is used to manage a cluster of EC2 instances, which run the service. It facilitates the deployment of tasks to these instances in a reliable manner. An [Elastic Load Balancer (ELB)](https://aws.amazon.com/elasticloadbalancing/) is attached to distribute traffic across the EC2 instances.
- [Docker Container Registry (ECR)](https://aws/amazon.com/ecr/) - ECR is used to store the docker images of the service. These images are referenced by ECS via task definitions, and deployed onto EC2 instances. 
- [ElastiCache](https://aws.amazon.com/elasticache/) - An ElastiCache node running Redis is used for quick storage and retrieval of a [Sorted Set](http://redis.io/topics/data-types).

Others:

- [Terraform](https://www.terraform.io/) - Terraform allows for complete deployment of Hello Service infrastructure via scripts. It is used to easily manage the state of infrastructure and modify it reliably and quickly. 
- [Docker](https://www.docker.com/) - A Docker image based on gliderlabs/alpine is used, allowing for consistency across development environments and painless deployments.
- [AWS Command Line Interface (AWS CLI)](https://aws.amazon.com/cli/) - AWS CLI commands are used to reliably deploy new revisions of the service.
- [Redis](http://redis.io/) - Redis is used for data storage in the local development environment. This offers parity with the ElastiCache node running Redis.


----------


## Local Development
Hello Service can be run locally on your machine via Docker - this greatly speeds up the process of testing new changes.
> **Note:**
> Hello Service has currently only been tested on OSX 10.11.5

### Requirements
The following must be set up locally:

- [Go](https://golang.org/doc/install)
- [Docker](https://docs.docker.com/engine/getstarted/step_one/)
- [Redis](http://redis.io/download)

### Running the Service
1. Navigate into the directory of the project source.
2. Run `make`. This will statically compile the service and subsequently build a docker image. Run this whenever changes are made to the source code of the Go project.
3. Run `redis-server`. This will initiate the local redis server. It may be desirable to run this on a separate terminal tab.
4. Run `docker run --publish 80:80 --env-file ./.env nytimes/hello`. This will run the service on localhost port 80. The environment variables outlined in the .env are loaded into the image. It may be necessary to confirm that the REDIS_CLUSTER_ADDRESS is accurate.

### Stopping the Service
1. Run `docker ps` to list the docker containers currently running.
2. Use `docker kill CONTAINER_ID` or `docker stop CONTAINER_ID` to stop the service.


----------

## Infrastructure Deployment
> **Note:**
> Infrastructure Deployment has currently only been tested on OSX 10.11.5

### Requirements
The following must be set up locally:

- [Terraform](https://www.terraform.io/) 

### Deploying the Infrastructure

 1. Navigate into the `deployment` directory of the project source.
 2. Run `terraform get`. This will install any terraform modules used.
 3. Run `terraform plan`. Follow the prompts. This will provide an overview of the infrastructure that will be deployed. Ensure that there are no errors before proceeding.
 4. Run `terraform apply`. Follow the prompts. This command will instantiate the infrastructure required to run Hello Service on AWS, and may take a few moments. 
 5. Run `terraform show`. This provides an overview of the deployed infrastructure. 
 6. Search for `aws_elasticache_cluster`. Add the value of `cache_nodes.0.address` to your clipboard.
 7. In the project source directory, create a file called `task_revision.json`, pasting the contents of `task_revision_example.json`. Paste the value in your clipboard into the REDIS_CLUSTER_ADDRESS.
 8. Use `terraform show` to obtain the `repository_url` from `aws_ecr_repository.hello-repository`. Update the appropriate value in the `task_revision.json` file. Do not include the leading `https://`. Setting up the `task_revision.json` file is necessary for deploying service updates in the future.
 9. The service will live at the address of the Load Balancer. Use `terraform show`, find `aws_elb.hello_service_elb`, and take the `dns_name`. Please note that a Release Deployment must be run for the actual service to run on the infrastructure, as the Docker image must be pushed to the repository, and the service updated with that image.
 
 > **Note:**
> The `terraform.tfstate` and `terraform.tfstate.backup` files that are generated should be kept (and likely committed to version control). These are necessary to iterate on the infrastructure of the service.



### Updating the Infrastructure
1. Make changes as desired to `deployment/infrastructure.tf`
2. Run `terraform plan`. This provides an overview of the resources that will be changed, added, and destroyed.
3. Run `terraform apply` to update the infrastructure. If infrastructure in the `infrastructure.tf` file has been deleted, these will be destroyed on AWS.

### Destroying the Infrastructure
1. Run `terraform plan -destroy`. This provides an overview of the resources that will be destroyed.
2. Run `terraform destroy` to destroy the infrastructure.

## Release Deployment
> **Note:**
> Release Deployment has currently only been tested on OSX 10.11.5

### Requirements
- [AWS Command Line Interface (AWS CLI)](https://aws.amazon.com/cli/) - AWS CLI commands are used to reliably deploy new revisions of the service.
- [Go](https://golang.org/doc/install)
- [Docker](https://docs.docker.com/engine/getstarted/step_one/)

### Updating the Service
1. Navigate into the directory of the project source. Ensure that the `task_revision.json` file is set up, as described in the Deploying the Infrastructure section
2. Run `make`. This will statically compile the service and subsequently build a docker image.
3. Run `docker tag nytimes/hello:latest YOUR_REPOSITORY_URL_HERE:latest`. Be sure to enter the URL of the ECR Repository.
4. Run `docker push YOUR_REPOSITORY_URL_HERE:latest`. If a response is received requesting to login, paste the commands provided, then attempt this step again.
5. Run `aws ecs register-task-definition --cli-input-json file://task_revision.json`. This will create a new task revision, which is necessary to update the service.
6. Run `aws ecs update-service --cluster ecs-hello --service hello-service --task-definition hello_service:REVISION_NUMBER`. ECS will run a blue-green deployment of the updates. This may take a while. Be sure to include the revision number from the output of step 5 into this command.


----------

## Endpoints


### hello/{name}

Outputs a hello message including the provided name, and updates a Sorted Set on Redis, containing the name and a score of how many times the name was provided to this endpoint.
#### Example Output

    Hello, "{name}"


### counts

Outputs a JSON structure containing the names that have been provided to the hello endpoint coupled with a count of how many times they were provided.

#### Example Output

    {
	  "counts": [
	    {
	      "name": "Bob",
	      "count": "1"
	    },
	    {
	      "name": "Bill",
	      "count": "3"
	    }
	  ]
	}


### health

Outputs a list of information about the server.

#### Example Output
On AWS:

	{
	  "ec2_instance_id": "i-4f066f57",
	  "uptime": 65473,
	  "cpu_utilization_percent": 0.27196008620621565,
	  "disk_utilization_percent": 43.82975528085612,
	  "total_ram_bytes_used": 95997952,
	  "total_ram_bytes_available": 947920896
	}
Local Machine:
	

    {
   	  "ec2_instance_id": "local",
   	  "uptime": 65473,
   	  "cpu_utilization_percent": 0.27196008620621565,
   	  "disk_utilization_percent": 43.82975528085612,
   	  "total_ram_bytes_used": 95997952,
   	  "total_ram_bytes_available": 947920896
   	}


### health/cluster

Outputs the health of all cluster instances that are **currently active on the load balancer**.

#### Example Output
On AWS:

    {
      "node_healths": [
        {
          "ec2_instance_id": "i-0236ca5d",
          "uptime": 65706,
          "cpu_utilization_percent": 0.33148680902462224,
          "disk_utilization_percent": 43.642749476192,
          "total_ram_bytes_used": 93929472,
          "total_ram_bytes_available": 949989376
        },
        {
          "ec2_instance_id": "i-4f066f57",
          "uptime": 65704,
          "cpu_utilization_percent": 0.16437408080283544,
          "disk_utilization_percent": 43.82975528085612,
          "total_ram_bytes_used": 96153600,
          "total_ram_bytes_available": 947765248
        }
      ]
    }
Local Machine:

    {
       "node_healths": [
         {
           "ec2_instance_id": "local",
           "uptime": 65706,
           "cpu_utilization_percent": 0.33148680902462224,
           "disk_utilization_percent": 43.642749476192,
           "total_ram_bytes_used": 93929472,
           "total_ram_bytes_available": 949989376
         }
       ]
     }


## What Next?
Improvements to consider for future iterations of Hello Service:

- Aggregation of deployment commands into a simple script. This could be a shell script, or any other scripting language using the AWS SDK.
- Log Centralization. It is currently necessary to SSH individually onto an instance to view the logs. Consider using the [ECS Log Collector](http://docs.aws.amazon.com/AmazonECS/latest/developerguide/troubleshooting.html#ecs-logs-collector).
- Redis connection management. If the connection to Redis is lost, this situation is not gracefully handled.
- An Alert system for critical errors. Consider using something like Pager Duty.
- Graceful JSON error responses
- Output ECR Repository URL as well as Redis Address from infrastructure.tf
