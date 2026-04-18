terraform {
  required_version = ">= 1.6.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

# ── VPC ──────────────────────────────────────────────────────────────────────

resource "aws_vpc" "inferflow" {
  cidr_block           = var.vpc_cidr
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = { Name = "${var.cluster_name}-vpc" }
}

resource "aws_internet_gateway" "inferflow" {
  vpc_id = aws_vpc.inferflow.id

  tags = { Name = "${var.cluster_name}-igw" }
}

resource "aws_subnet" "inferflow" {
  count             = length(var.availability_zones)
  vpc_id            = aws_vpc.inferflow.id
  cidr_block        = cidrsubnet(var.vpc_cidr, 4, count.index)
  availability_zone = var.availability_zones[count.index]

  map_public_ip_on_launch = true

  tags = {
    Name                                        = "${var.cluster_name}-subnet-${count.index}"
    "kubernetes.io/cluster/${var.cluster_name}" = "shared"
    "kubernetes.io/role/elb"                    = "1"
  }
}

resource "aws_route_table" "inferflow" {
  vpc_id = aws_vpc.inferflow.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.inferflow.id
  }

  tags = { Name = "${var.cluster_name}-rt" }
}

resource "aws_route_table_association" "inferflow" {
  count          = length(var.availability_zones)
  subnet_id      = aws_subnet.inferflow[count.index].id
  route_table_id = aws_route_table.inferflow.id
}

# ── IAM: EKS Cluster ─────────────────────────────────────────────────────────

resource "aws_iam_role" "cluster" {
  name = "${var.cluster_name}-cluster-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "eks.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "cluster_policy" {
  role       = aws_iam_role.cluster.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
}

# ── IAM: Node Groups ─────────────────────────────────────────────────────────

resource "aws_iam_role" "node" {
  name = "${var.cluster_name}-node-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "ec2.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "node_policy" {
  role       = aws_iam_role.node.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy"
}

resource "aws_iam_role_policy_attachment" "node_cni_policy" {
  role       = aws_iam_role.node.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy"
}

resource "aws_iam_role_policy_attachment" "node_ecr_policy" {
  role       = aws_iam_role.node.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
}

# ── EKS Cluster ──────────────────────────────────────────────────────────────

resource "aws_eks_cluster" "inferflow" {
  name     = var.cluster_name
  role_arn = aws_iam_role.cluster.arn
  version  = var.kubernetes_version

  vpc_config {
    subnet_ids = aws_subnet.inferflow[*].id
  }

  depends_on = [aws_iam_role_policy_attachment.cluster_policy]
}

# ── Node Group: System ───────────────────────────────────────────────────────

resource "aws_eks_node_group" "system" {
  cluster_name    = aws_eks_cluster.inferflow.name
  node_group_name = "system"
  node_role_arn   = aws_iam_role.node.arn
  subnet_ids      = aws_subnet.inferflow[*].id
  instance_types  = [var.system_node_size]

  scaling_config {
    desired_size = var.system_node_count
    min_size     = 1
    max_size     = 2
  }

  labels = { "inferflow/node-pool" = "system" }

  disk_size = 50

  depends_on = [
    aws_iam_role_policy_attachment.node_policy,
    aws_iam_role_policy_attachment.node_cni_policy,
    aws_iam_role_policy_attachment.node_ecr_policy,
  ]
}

# ── Node Group: vLLM ─────────────────────────────────────────────────────────

resource "aws_eks_node_group" "vllm" {
  cluster_name    = aws_eks_cluster.inferflow.name
  node_group_name = "vllm"
  node_role_arn   = aws_iam_role.node.arn
  subnet_ids      = aws_subnet.inferflow[*].id
  instance_types  = [var.vllm_node_size]

  scaling_config {
    desired_size = var.vllm_node_count
    min_size     = 1
    max_size     = 3
  }

  labels = { "inferflow/node-pool" = "vllm" }

  disk_size = 50

  depends_on = [
    aws_iam_role_policy_attachment.node_policy,
    aws_iam_role_policy_attachment.node_cni_policy,
    aws_iam_role_policy_attachment.node_ecr_policy,
  ]
}

# ── ECR ──────────────────────────────────────────────────────────────────────

resource "aws_ecr_repository" "inferflow" {
  name                 = var.registry_name
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = false
  }
}
