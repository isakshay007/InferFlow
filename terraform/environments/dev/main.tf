terraform {
  required_version = ">= 1.5.0"
}

provider "aws" {
  region = var.aws_region
}

locals {
  name = "inferflow-dev"
  tags = {
    Project     = "InferFlow"
    Environment = "dev"
    ManagedBy   = "terraform"
  }
}

module "network" {
  source = "../../modules/network"

  name            = local.name
  vpc_cidr        = var.vpc_cidr
  public_subnets  = var.public_subnets
  private_subnets = var.private_subnets
  tags            = local.tags
}

module "eks" {
  source = "../../modules/eks"

  name                = local.name
  cluster_name        = var.cluster_name
  kubernetes_version  = var.kubernetes_version
  private_subnet_ids  = module.network.private_subnet_ids
  node_instance_types = var.node_instance_types
  desired_size        = var.desired_size
  min_size            = var.min_size
  max_size            = var.max_size
  tags                = local.tags
}
