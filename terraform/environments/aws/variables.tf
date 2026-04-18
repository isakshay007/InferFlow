variable "region" {
  description = "AWS region."
  type        = string
  default     = "us-east-1"
}

variable "cluster_name" {
  description = "EKS cluster name."
  type        = string
  default     = "inferflow-eks"
}

variable "kubernetes_version" {
  description = "EKS Kubernetes version."
  type        = string
  default     = "1.32"
}

variable "vpc_cidr" {
  description = "CIDR block for the VPC."
  type        = string
  default     = "10.10.0.0/16"
}

variable "availability_zones" {
  description = "Availability zones for subnets."
  type        = list(string)
  default     = ["us-east-1a", "us-east-1b"]
}

variable "system_node_size" {
  description = "Instance type for the system node group (router + Redis)."
  type        = string
  default     = "t3.medium"
}

variable "system_node_count" {
  description = "Number of nodes in the system group."
  type        = number
  default     = 1
}

variable "vllm_node_size" {
  description = "Instance type for the vLLM node group."
  type        = string
  default     = "c5.xlarge"
}

variable "vllm_node_count" {
  description = "Number of nodes in the vLLM group."
  type        = number
  default     = 3
}

variable "registry_name" {
  description = "ECR repository name."
  type        = string
  default     = "inferflow"
}
