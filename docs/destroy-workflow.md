# Destroy Infrastructure

To remove the active AWS infrastructure, destroy it manually from Terraform.

## Terraform Destroy

```bash
cd terraform/environments/aws
terraform init -backend=false
terraform destroy
```

## What This Removes

- EKS cluster
- system node group
- worker node group
- AWS VPC and subnets
- ECR repository

## Notes

- The Terraform state is local for the EKS environment, so run `destroy` from the same workspace you used for `apply`.
- Delete any leftover container images in ECR if you want a completely clean project.
