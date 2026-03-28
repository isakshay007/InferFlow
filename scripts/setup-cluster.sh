#!/usr/bin/env bash
set -euo pipefail

echo "InferFlow cluster bootstrap"
echo "This Week 1 script provisions AWS baseline infrastructure with Terraform and leaves in-cluster deployment to Helm/kubectl."
echo "Before running, export AWS credentials and review terraform/environments/dev/terraform.tfvars."

cd terraform/environments/dev
terraform init
terraform apply

echo "Next steps after Terraform apply:"
echo "1. Update kubeconfig for the created EKS cluster."
echo "2. Install the NVIDIA device plugin on the cluster if it is not already present."
echo "3. Build and publish the router, triton-adapter, and triton-qwen3 images."
echo "4. Deploy k8s/triton.yaml, k8s/triton-adapter.yaml, then k8s/router.yaml."
echo "5. Verify one end-to-end request through the router before moving on."
