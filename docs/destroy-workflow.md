# Destroy Workflow

InferFlow includes a manual-only AWS cleanup workflow.

Workflow file:

- [destroy-aws.yml](../.github/workflows/destroy-aws.yml)

## What It Does

- deletes Kubernetes app resources
- destroys Terraform-managed AWS infrastructure
- deletes ECR repositories
- optionally destroys the shared Terraform state bucket and DynamoDB lock table

## Safety Controls

- the workflow is manual only
- you must type `DESTROY`
- `destroy_state_backend` should stay `false` unless you want a full wipe

## Full Wipe

To destroy everything created by the automated setup:

- run the destroy workflow manually
- set `confirm_destroy` to `DESTROY`
- set `destroy_state_backend` to `true`
