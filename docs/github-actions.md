# GitHub Actions

InferFlow uses GitHub Actions for CI and AWS Terraform planning.

## Workflows

- [ci.yml](C:/Users/ajinf/Documents/CS%206650/InferFlow/.github/workflows/ci.yml)
- [terraform-plan.yml](C:/Users/ajinf/Documents/CS%206650/InferFlow/.github/workflows/terraform-plan.yml)

## Required Repository Secrets

- `AWS_ACCESS_KEY_ID` — AWS access key ID with EKS and ECR permissions
- `AWS_SECRET_ACCESS_KEY` — AWS secret access key corresponding to the access key ID

## Recommended Order

1. Let CI pass.
2. Review the Terraform Plan for EKS.
3. Run Terraform apply manually from your local machine.
4. Build and deploy the router, Redis, and vLLM manifests to the created cluster.

## Notes

- The Terraform plan workflow targets [terraform/environments/aws](C:/Users/ajinf/Documents/CS%206650/InferFlow/terraform/environments/aws).
- Container image push and cluster deployment are currently manual for the EKS path.
