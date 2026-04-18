# Terraform Infrastructure

Terraform manages the AWS baseline for InferFlow:

- EKS cluster
- system node group for router, Redis, and observability components
- worker node group for vLLM workers
- AWS VPC and subnets
- ECR repository

## Current Recommended Defaults

- cluster name: `inferflow-eks`
- region: `us-east-1`
- system node group: `t3.medium` with `2` nodes
- worker node group: `c5.xlarge`
- registry name: `inferflow`

## Layout

Main environment files:

- [main.tf](C:/Users/ajinf/Documents/CS%206650/InferFlow/terraform/environments/aws/main.tf)
- [variables.tf](C:/Users/ajinf/Documents/CS%206650/InferFlow/terraform/environments/aws/variables.tf)
- [outputs.tf](C:/Users/ajinf/Documents/CS%206650/InferFlow/terraform/environments/aws/outputs.tf)
- [terraform.tfvars.example](C:/Users/ajinf/Documents/CS%206650/InferFlow/terraform/environments/aws/terraform.tfvars.example)

## Usage

```bash
cd terraform/environments/aws
terraform init -backend=false
terraform validate
terraform plan
terraform apply
```

## Notes

- Authentication uses `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` set in the environment; no project ID variable is required.
- The router and Redis manifests target nodes labeled `inferflow/node-pool=system`.
- The vLLM worker manifest targets nodes labeled `inferflow/node-pool=worker`.
- After apply, configure kubectl with: `aws eks update-kubeconfig --region us-east-1 --name inferflow-eks`
