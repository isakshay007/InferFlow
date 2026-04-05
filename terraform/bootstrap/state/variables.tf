variable "aws_region" {
  type    = string
  default = "us-east-1"
}

variable "state_bucket_name" {
  type    = string
  default = "inferflow-tfstate"
}

variable "lock_table_name" {
  type    = string
  default = "inferflow-tfstate-lock"
}

variable "tags" {
  type    = map(string)
  default = {}
}
