# Terraform

Terraform owns the AWS baseline for InferFlow.

## Managed Resources

- AWS VPC and subnets
- EKS cluster
- system node group
- worker node group
- ECR repository

## Main Environment

Use the AWS environment:

- [terraform/environments/aws/main.tf](environments/aws/main.tf)

## Current Defaults

- cluster name: `inferflow-eks`
- region: `us-east-1`
- system node group: `t3.medium` with `2` nodes
- worker node group: `c5.xlarge`
- registry name: `inferflow`

## Usage

```bash
cd terraform/environments/aws
terraform init -backend=false
terraform validate
terraform plan
terraform apply
```

After apply, configure kubectl:

```bash
aws eks update-kubeconfig --region us-east-1 --name inferflow-eks
```

## GitHub Actions

The Terraform plan workflow expects:

- repository secret `AWS_ACCESS_KEY_ID`
- repository secret `AWS_SECRET_ACCESS_KEY`
