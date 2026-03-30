# Terraform Infrastructure

Terraform manages the AWS baseline for InferFlow:

- VPC and subnets
- EKS cluster
- GPU node group
- IAM roles and policy attachments

## Shared Remote State

InferFlow uses:

- S3 for Terraform state
- DynamoDB for state locking

Main backend file:

- [backend.tf](../terraform/environments/dev/backend.tf)

## One-Time Bootstrap

Bootstrap the shared state resources once:

```bash
cd terraform/bootstrap/state
terraform init -backend=false
terraform apply
```

Files:

- [bootstrap main.tf](../terraform/bootstrap/state/main.tf)
- [bootstrap tfvars example](../terraform/bootstrap/state/terraform.tfvars.example)

## Main Environment Usage

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
terraform apply
```

Recommended state key:

- `inferflow/dev/terraform.tfstate`
