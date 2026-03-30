# GitHub Actions

InferFlow uses GitHub Actions for CI, infrastructure, deploy, and cleanup.

## Workflows

- [ci.yml](../.github/workflows/ci.yml)
- [terraform-plan.yml](../.github/workflows/terraform-plan.yml)
- [terraform-apply.yml](../.github/workflows/terraform-apply.yml)
- [deploy-aws-triton.yml](../.github/workflows/deploy-aws-triton.yml)
- [destroy-aws.yml](../.github/workflows/destroy-aws.yml)

## Required Repository Variables

- `AWS_REGION`
- `EKS_CLUSTER_NAME`
- `ECR_REPOSITORY_PREFIX`
- `TF_STATE_BUCKET`
- `TF_LOCK_TABLE`
- `TF_STATE_KEY`

## Required Repository Secret

- `AWS_ROLE_TO_ASSUME`

## Recommended Order

1. Let CI pass.
2. Review Terraform Plan.
3. Run Terraform Apply manually.
4. Run Deploy AWS Triton Stack manually.

## ECR Behavior

The deploy workflow creates these ECR repositories only if they do not already exist:

- `<prefix>/router`
- `<prefix>/triton-adapter`
- `<prefix>/triton-qwen3`
