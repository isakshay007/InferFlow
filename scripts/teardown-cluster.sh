#!/usr/bin/env bash
set -euo pipefail

echo "InferFlow EKS teardown"
echo "This destroys the Terraform-managed AWS baseline for the EKS environment."

cd terraform/environments/aws
terraform init -backend=false
terraform destroy
