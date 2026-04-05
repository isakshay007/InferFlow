terraform {
  backend "s3" {
    bucket         = "inferflow-tfstate"
    key            = "dev/terraform.tfstate"
    region         = "us-east-1"
    dynamodb_table = "inferflow-tfstate-lock"
    encrypt        = true
  }
}
