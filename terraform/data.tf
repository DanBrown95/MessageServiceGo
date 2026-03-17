data "aws_security_group" "reroll_drinks_default" { ## Id for the default security group
  name = "default"
  vpc_id = var.vpc_id
}

data "aws_subnet" "subnet_a" {
  filter {
    name   = "tag:Name"
    values = ["us-east-2a"]
  }
}

data "aws_subnet" "subnet_b" {
  filter {
    name   = "tag:Name"
    values = ["us-east-2b"]
  }
}

data "aws_subnet" "subnet_c" {
  filter {
    name   = "tag:Name"
    values = ["us-east-2c"]
  }
}
