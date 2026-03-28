# GitHub Actions

InferFlow uses GitHub Actions for CI, infrastructure, deploy, and cleanup.

## Workflows

- [ci.yml](C:/Users/ajinf/Documents/CS%206650/InferFlow/.github/workflows/ci.yml)
- [terraform-plan.yml](C:/Users/ajinf/Documents/CS%206650/InferFlow/.github/workflows/terraform-plan.yml)
- [terraform-apply.yml](C:/Users/ajinf/Documents/CS%206650/InferFlow/.github/workflows/terraform-apply.yml)
- [deploy-aws-triton.yml](C:/Users/ajinf/Documents/CS%206650/InferFlow/.github/workflows/deploy-aws-triton.yml)
- [destroy-aws.yml](C:/Users/ajinf/Documents/CS%206650/InferFlow/.github/workflows/destroy-aws.yml)

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
