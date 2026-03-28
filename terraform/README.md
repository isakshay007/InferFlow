# Terraform

Terraform owns the AWS baseline for InferFlow:

- VPC and subnets
- EKS control plane
- Managed node group
- IAM roles and policy attachments

It now uses a shared remote backend for team-safe state management:

- S3 bucket for Terraform state
- DynamoDB table for state locking

## One-Time Remote State Bootstrap

Bootstrap the backend resources once from `terraform/bootstrap/state` using local state:

```bash
cd terraform/bootstrap/state
terraform init -backend=false
terraform apply
```

This creates:

- the S3 bucket that stores the Terraform state file
- the DynamoDB table that provides state locking

See [terraform/bootstrap/state/terraform.tfvars.example](/C:/Users/ajinf/Documents/CS%206650/InferFlow/terraform/bootstrap/state/terraform.tfvars.example) for the required inputs.

## Shared Backend Usage

After the bucket and lock table exist, initialize the main environment with remote state:

```bash
cd terraform/environments/dev
terraform init \
  -backend-config="bucket=<state-bucket-name>" \
  -backend-config="key=<state-key>" \
  -backend-config="region=<aws-region>" \
  -backend-config="dynamodb_table=<lock-table-name>" \
  -backend-config="encrypt=true"
terraform validate
terraform plan
```

Recommended state key for the dev environment:

- `inferflow/dev/terraform.tfstate`

## GitHub Actions Variables

The Terraform plan/apply workflows expect these repository variables:

- `AWS_REGION`
- `TF_STATE_BUCKET`
- `TF_LOCK_TABLE`
- `TF_STATE_KEY`

The workflows also expect this repository secret:

- `AWS_ROLE_TO_ASSUME`
