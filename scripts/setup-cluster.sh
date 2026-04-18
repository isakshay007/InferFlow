#!/usr/bin/env bash
set -euo pipefail

echo "InferFlow EKS bootstrap"
echo "This script provisions the AWS baseline with Terraform and leaves in-cluster deployment to kubectl."
echo "Before running, configure AWS credentials via 'aws configure' and review terraform/environments/aws/terraform.tfvars.example."

cd terraform/environments/aws
terraform init -backend=false
terraform apply

echo "Next steps after Terraform apply:"
echo "1. Fetch kubeconfig: aws eks update-kubeconfig --name inferflow-eks --region us-east-1"
echo "2. Build and publish the router and vllm-adapter images to ECR."
echo "3. Deploy k8s/redis.yaml, k8s/vllm-worker.yaml, k8s/router.yaml, then k8s/keda-vllm.yaml."
echo "4. Verify one end-to-end request through the router before running experiments."
