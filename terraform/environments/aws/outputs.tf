output "cluster_name" {
  value       = aws_eks_cluster.inferflow.name
  description = "EKS cluster name."
}

output "cluster_endpoint" {
  value       = aws_eks_cluster.inferflow.endpoint
  description = "Kubernetes API endpoint."
}

output "cluster_version" {
  value       = aws_eks_cluster.inferflow.version
  description = "EKS cluster Kubernetes version."
}

output "registry_endpoint" {
  value       = aws_ecr_repository.inferflow.repository_url
  description = "ECR repository URL."
}

output "vpc_id" {
  value       = aws_vpc.inferflow.id
  description = "AWS VPC ID."
}
