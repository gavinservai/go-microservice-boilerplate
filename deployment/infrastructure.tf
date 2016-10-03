provider "aws" {
  access_key = "${var.access_key}"
  secret_key = "${var.secret_key}"
  region = "${var.region}"
}

module "vpc" {
  source = "github.com/terraform-community-modules/tf_aws_vpc"
  name = "hello-service-vpc"
  cidr = "10.0.0.0/16"
  public_subnets  = ["10.0.101.0/24","10.0.102.0/24"]
  azs = ["us-west-2a", "us-west-2b"]
}

resource "aws_security_group" "allow_all_outbound" {
  name_prefix = "hello-service-"
  description = "Allow all outbound traffic"
  vpc_id = "${module.vpc.vpc_id}"

  egress = {
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group" "allow_all_inbound" {
  name_prefix = "hello-service-"
  description = "Allow all inbound traffic"
  vpc_id = "${module.vpc.vpc_id}"

  ingress = {
    from_port = 0
    to_port = 0
    protocol = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group" "allow_cluster" {
  name_prefix = "hello-service-"
  description = "Allow all traffic within cluster"
  vpc_id = "${module.vpc.vpc_id}"

  ingress = {
    from_port = 0
    to_port = 65535
    protocol = "tcp"
    self = true
  }

  egress = {
    from_port = 0
    to_port = 65535
    protocol = "tcp"
    self = true
  }
}

# Load Balancer for ECS
resource "aws_elb" "hello-service-balancer" {
    name = "hello-service-balancer"
    subnets = ["${module.vpc.public_subnets}"]
    security_groups = [
      "${aws_security_group.allow_cluster.id}",
      "${aws_security_group.allow_all_inbound.id}",
      "${aws_security_group.allow_all_outbound.id}"
    ]

    listener {
        lb_port = 80
        lb_protocol = "http"
        instance_port = 80
        instance_protocol = "http"
    }
}

# ECS Cluster
resource "aws_ecs_cluster" "hello-service-cluster" {
    name = "hello-service-cluster"
}

# Auto-Scaling Group for Cluster
resource "aws_autoscaling_group" "hello-service-cluster-instances" {
  name = "hello-service-cluster-instances"
  vpc_zone_identifier = ["${module.vpc.public_subnets}"]
  min_size = 3
  max_size = 5
  launch_configuration = "${aws_launch_configuration.hello-service-instance.name}"
}

# Profile for ECS EC2 instance
resource "aws_iam_instance_profile" "ecs-iam-profile" {
  name = "ecs-iam-profile"
  roles = ["${aws_iam_role.ecs_role.name}"]
}

# Instance Configuration for Cluster
resource "aws_launch_configuration" "hello-service-instance" {
  name_prefix = "hello-service-instance-"
  instance_type = "t2.micro"
  image_id = "ami-1ccd1f7c"
  iam_instance_profile = "${aws_iam_instance_profile.ecs-iam-profile.id}"
  security_groups = [
    "${aws_security_group.allow_all_outbound.id}",
    "${aws_security_group.allow_cluster.id}",
  ]
  associate_public_ip_address = true
}

# Docker Image Repository for ECS
resource "aws_ecr_repository" "nytimes-hello-repository" {
  name = "nytimes-hello-repository"
}

# ECS Task
resource "aws_ecs_task_definition" "hello-service-task" {
  family = "hello-service-task"
  container_definitions = <<EOF
  [
    {
      "name": "hello-service-task",
      "image": "${aws_ecr_repository.nytimes-hello-repository.repository_url}:latest",
      "cpu": 10,
      "memory": 500,
      "essential": true,
      "portMappings": [
        {
          "containerPort": 80,
          "hostPort": 80
        }
      ],
      "environment": [{
        "name": "REDIS_CLUSTER_ADDRESS",
        "value": "${aws_elasticache_cluster.hello-service-cache.cache_nodes.0.address}:${aws_elasticache_cluster.hello-service-cache.cache_nodes.0.port}"
      }]
    }
  ]
EOF
}

# IAM Role, necessary to use Balancer
resource "aws_iam_role" "ecs_role" {
    name = "nytimes_hello_role"
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


# Policy Attaching EC2 for the ECS Role
resource "aws_iam_policy_attachment" "ec2_for_ecs_role" {
    name = "ec2_for_ecs_role"
    roles = ["${aws_iam_role.ecs_role.id}"]
    policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2ContainerServiceforEC2Role"
}

# Role for Load Balancer
resource "aws_iam_role" "ecs_elb_role" {
  name = "ecs_elb_role"
  assume_role_policy = <<EOF
{
    "Version": "2012-10-17",
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

# Policy Attaching EC2 for ELB Role
resource "aws_iam_policy_attachment" "ec2_for_ecs_elb_role" {
  name = "ec2_for_ecs_elb_role"
  roles = ["${aws_iam_role.ecs_elb_role.id}"]
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2ContainerServiceRole"
}

# ECS Service with Balancer
resource "aws_ecs_service" "hello-service" {
  name = "hello-service"
  cluster = "${aws_ecs_cluster.hello-service-cluster.id}"
  task_definition = "${aws_ecs_task_definition.hello-service-task.arn}"
  desired_count = 5
  iam_role = "${aws_iam_role.ecs_elb_role.arn}"
  depends_on = ["aws_iam_policy_attachment.ec2_for_ecs_elb_role"]

  load_balancer {
      elb_name = "${aws_elb.hello-service-balancer.id}"
      container_name = "hello-service-task"
      container_port = 80
  }
}

# Elasticache Cluster
resource "aws_elasticache_cluster" "hello-service-cache" {
    cluster_id = "hello-cache"
    engine = "redis"
    node_type = "cache.t2.micro"
    port = 6379
    num_cache_nodes = 1
}

output "docker repository" {
    value = "${aws_ecr_repository.nytimes-hello-repository.repository_url}"
}