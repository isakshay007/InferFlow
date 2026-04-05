variable "aws_region" {
  type    = string
  default = "us-east-1"
}

variable "cluster_name" {
  type    = string
  default = "inferflow-dev"
}

variable "kubernetes_version" {
  type    = string
  default = "1.29"
}

variable "vpc_cidr" {
  type    = string
  default = "10.42.0.0/16"
}

variable "public_subnets" {
  type = map(object({
    cidr              = string
    availability_zone = string
  }))
  default = {
    a = {
      cidr              = "10.42.1.0/24"
      availability_zone = "us-east-1a"
    }
    b = {
      cidr              = "10.42.2.0/24"
      availability_zone = "us-east-1b"
    }
  }
}

variable "private_subnets" {
  type = map(object({
    cidr              = string
    availability_zone = string
  }))
  default = {
    a = {
      cidr              = "10.42.10.0/24"
      availability_zone = "us-east-1a"
    }
    b = {
      cidr              = "10.42.11.0/24"
      availability_zone = "us-east-1b"
    }
  }
}

variable "node_instance_types" {
  type    = list(string)
  default = ["c5.large"]
}

variable "desired_size" {
  type    = number
  default = 3
}

variable "min_size" {
  type    = number
  default = 1
}

variable "max_size" {
  type    = number
  default = 4
}
