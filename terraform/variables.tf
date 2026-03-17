variable "aws_region" {
  description = "The AWS region where resources will be created"
  default     = "us-east-2"
}

variable "service_name" {
  description = "The name of the service"
  default     = "MessageService"
}

variable "account_id" {
  description = "The aws account id"
  default     = "952042308140"
}

variable "vpc_id" {
  default = "vpc-3b3a8f50"
}
