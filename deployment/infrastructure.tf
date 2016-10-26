provider "aws" {
  access_key = "${var.access_key}"
  secret_key = "${var.secret_key}"
  region = "${var.region}"
}

# VPC for our infrastructure
module "vpc" {
    source = "github.com/terraform-community-modules/tf_aws_vpc"
    enable_dns_support = true
    name = "ecs-vpc"
    cidr = "10.0.0.0/16"
    public_subnets  = ["10.0.101.0/24", "10.0.102.0/24"]
    azs = ["us-west-2a", "us-west-2b"]
}

# Security group that allows all outbound traffic
resource "aws_security_group" "allow_all_outbound" {
    name_prefix = "${module.vpc.vpc_id}-"
    description = "Allow all outbound traffic"
    vpc_id = "${module.vpc.vpc_id}"

    egress = {
        from_port = 0
        to_port = 0
        protocol = "-1"
        cidr_blocks = ["0.0.0.0/0"]
    }
}

# Security group that allows all inbound traffic
resource "aws_security_group" "allow_all_inbound" {
    name_prefix = "${module.vpc.vpc_id}-"
    description = "Allow all inbound traffic"
    vpc_id = "${module.vpc.vpc_id}"

    ingress = {
        from_port = 0
        to_port = 0
        protocol = "-1"
        cidr_blocks = ["0.0.0.0/0"]
    }
}

# Security group that allows all traffic between clusters
resource "aws_security_group" "allow_cluster" {
    name_prefix = "${module.vpc.vpc_id}-"
    description = "Allow all traffic within cluster"
    vpc_id = "${module.vpc.vpc_id}"

    ingress = {
        from_port = 0
        to_port = 65535
        protocol = "tcp"
        self = true
        cidr_blocks = ["10.0.0.0/16"]
    }

    egress = {
        from_port = 0
        to_port = 65535
        protocol = "tcp"
        self = true
    }
}

# Security group that allows all inbound SSH traffic
resource "aws_security_group" "allow_all_ssh" {
    name_prefix = "${module.vpc.vpc_id}-"
    description = "Allow all inbound SSH traffic"
    vpc_id = "${module.vpc.vpc_id}"

    ingress = {
        from_port = 22
        to_port = 22
        protocol = "tcp"
        cidr_blocks = ["0.0.0.0/0"]
    }
}

# Security group that allows connections for redis
resource "aws_security_group" "redis" {
    name_prefix = "${module.vpc.vpc_id}-"
    vpc_id = "${module.vpc.vpc_id}"

    ingress = {
        from_port = 0
        to_port = 6379
        protocol = "tcp"
        self = true
    }

    egress = {
        from_port = 0
        to_port = 6379
        protocol = "tcp"
        self = true
    }
}

# Role to be assumed by the container service
resource "aws_iam_role" "ecs" {
    name = "ecs"
    assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": "sts:AssumeRole",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Effect": "Allow",
      "Sid": ""
    }
  ]
}
EOF
}

# Custom policy giving instance permissions to perform cluster health check
resource "aws_iam_policy" "policy" {
    name = "ecs_policy"
    path = "/"
    description = "Used by container instance to query for other container instances"
    policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Action": [
        "elasticloadbalancing:DescribeLoadBalancers",
        "ec2:DescribeInstances"
      ],
      "Effect": "Allow",
      "Resource": "*"
    }
  ]
}
EOF
}

# Attaches ecs policy to the ecs role
resource "aws_iam_policy_attachment" "ecs_for_ec2" {
    name = "ecs-for-ec2"
    roles = ["${aws_iam_role.ecs.id}"]
    policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2ContainerServiceforEC2Role"
}

# Attaches custom policy to the ecs role
resource "aws_iam_policy_attachment" "ecs_health" {
    name = "ecs-health"
    roles = ["${aws_iam_role.ecs.id}"]
    policy_arn = "${aws_iam_policy.policy.arn}"
}

# Role for the service load balancer
resource "aws_iam_role" "ecs_elb" {
    name = "ecs-elb"
    assume_role_policy = <<EOF
{
  "Version": "2008-10-17",
  "Statement": [
    {
      "Sid": "",
      "Effect": "Allow",
      "Principal": {
        "Service": "ecs.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

# Attaches ecs policy to the ecs_elb role
resource "aws_iam_policy_attachment" "ecs_elb" {
    name = "ecs_elb"
    roles = ["${aws_iam_role.ecs_elb.id}"]
    policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2ContainerServiceRole"
}

# ECS cluster
resource "aws_ecs_cluster" "hello" {
    name = "ecs-hello"
}

# Repository to store docker images of service
resource "aws_ecr_repository" "hello-repository" {
  name = "hello-repository"
}

# Task definition for the service
resource "aws_ecs_task_definition" "hello_service" {
    family = "hello_service"
    container_definitions = <<EOF
    [
      {
        "name": "hello-service",
        "image": "${aws_ecr_repository.hello-repository.registry_id}.dkr.ecr.${var.region}.amazonaws.com/${aws_ecr_repository.hello-repository.name}:latest",
        "cpu": 0,
        "memory": 128,
        "essential": true,
        "environment": [
          {
              "name": "REDIS_CLUSTER_ADDRESS",
              "value": "${aws_elasticache_cluster.hello_redis_cluster.cache_nodes.0.address}:${aws_elasticache_cluster.hello_redis_cluster.cache_nodes.0.port}"
          },
          {
              "name": "ELB_NAME",
              "value": "${aws_elb.hello_service_elb.name}"
          }
        ],
        "portMappings": [
          {
            "containerPort": 80,
            "hostPort": 80
          }
        ]
      }
    ]
EOF
}

# Load balancer for ECS instances
resource "aws_elb" "hello_service_elb" {
    name = "hello-service-elb"
    subnets = ["${module.vpc.public_subnets}"]
    connection_draining = true
    cross_zone_load_balancing = true
    security_groups = [
        "${aws_security_group.allow_cluster.id}",
        "${aws_security_group.allow_all_inbound.id}",
        "${aws_security_group.allow_all_outbound.id}"
    ]

    listener {
        instance_port = 80
        instance_protocol = "http"
        lb_port = 80
        lb_protocol = "http"
    }

    health_check {
        healthy_threshold = 2
        unhealthy_threshold = 10
        target = "HTTP:80/"
        interval = 5
        timeout = 4
    }
}

# ECS Service
resource "aws_ecs_service" "hello_service" {
    name = "hello-service"
    cluster = "${aws_ecs_cluster.hello.id}"
    task_definition = "${aws_ecs_task_definition.hello_service.arn}"
    desired_count = 2
    iam_role = "${aws_iam_role.ecs_elb.arn}"
    depends_on = ["aws_iam_policy_attachment.ecs_elb"]

    load_balancer {
        elb_name = "${aws_elb.hello_service_elb.id}"
        container_name = "hello-service"
        container_port = 80
    }
}

resource "template_file" "user_data" {
    template = "templates/user_data"
    vars {
        cluster_name = "ecs-hello"
    }
}

# Instance profile for the launch configuration
resource "aws_iam_instance_profile" "ecs" {
    name = "ecs-profile"
    roles = ["${aws_iam_role.ecs.name}"]
}

# The launch configuration for an instance
resource "aws_launch_configuration" "ecs_cluster" {
    name = "ecs_cluster_conf"
    instance_type = "t2.micro"
    image_id = "${lookup(var.ami, var.region)}"
    iam_instance_profile = "${aws_iam_instance_profile.ecs.id}"
    security_groups = [
        "${aws_security_group.allow_all_ssh.id}",
        "${aws_security_group.allow_all_outbound.id}",
        "${aws_security_group.allow_cluster.id}",
        "${aws_security_group.redis.id}"
    ]
    user_data = "${template_file.user_data.rendered}"
    key_name = "aws-eb"
}

# Autoscaling group for the cluster. Currently set to a fixed number
resource "aws_autoscaling_group" "ecs_cluster" {
    name = "ecs-cluster"
    vpc_zone_identifier = ["${module.vpc.public_subnets}"]
    min_size = 0
    max_size = 3
    desired_capacity = 3
    launch_configuration = "${aws_launch_configuration.ecs_cluster.name}"
    health_check_type = "EC2"
}

# Subnet group for elasticache
resource "aws_elasticache_subnet_group" "elasticache_subnet" {
  name = "elasticache-subnet"
  subnet_ids = ["${module.vpc.public_subnets}"]
}

# Elasticache cluster running redis
resource "aws_elasticache_cluster" "hello_redis_cluster" {
    cluster_id = "hello-cache"
    engine = "redis"
    node_type = "cache.t2.micro"
    port = 6379
    num_cache_nodes = 1
    subnet_group_name = "${aws_elasticache_subnet_group.elasticache_subnet.name}"
    security_group_ids = ["${aws_security_group.redis.id}"]
}

variable "ami" {
    description = "AWS ECS AMI id"
    default = {
        us-east-1 = "ami-cb2305a1"
        us-west-1 = "ami-bdafdbdd"
        us-west-2 = "ami-ec75908c"
        eu-west-1 = "ami-13f84d60"
        eu-central-1 =  "ami-c3253caf"
        ap-northeast-1 = "ami-e9724c87"
        ap-southeast-1 = "ami-5f31fd3c"
        ap-southeast-2 = "ami-83af8ae0"
    }
}