provider "aws" {
  region = var.aws_region
}

## -- Create Users and roles
resource "aws_iam_user" "devops_user" { # User to attach to circleci
  name = "${var.service_name}-devops"
}

resource "aws_iam_user_policy_attachment" "ecs_full_access" {
  user       = aws_iam_user.devops_user.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryFullAccess"
}

resource "aws_iam_user_policy_attachment" "ecs_power_user" {
  user       = aws_iam_user.devops_user.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryPowerUser"
}


resource "aws_iam_user_policy" "ecs_custom_policy" {
  name = "ECR-ECS-Policy"
  user = aws_iam_user.devops_user.name
  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Effect = "Allow",
        Action = [
          "ecr:GetAuthorizationToken",
          "ecr:BatchCheckLayerAvailability",
          "ecr:BatchGetImage",
          "ecr:InitiateLayerUpload",
          "ecr:UploadLayerPart",
          "ecr:CompleteLayerUpload",
          "ecr:PutImage"
        ],
        Resource = ["*"]
      },
      {
        Effect = "Allow",
        Action = [
          "ecs:RegisterTaskDefinition",
          "ecs:UpdateService"
        ],
        Resource = [
          "arn:aws:ecs:${var.aws_region}:${var.account_id}:service/${var.service_name}/*",
          "arn:aws:ecs:${var.aws_region}:${var.account_id}:task-definition/${var.service_name}-*"
        ]
      }
    ]
  })
}

resource "aws_iam_role" "container_role" { ## role for the ec2 containers
  name = var.service_name
  assume_role_policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Effect = "Allow",
        Principal = {
          Service = "ec2.amazonaws.com"
        },
        Action = "sts:AssumeRole"
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "ecs_read_only" {
  role       = aws_iam_role.container_role.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
}

resource "aws_iam_role_policy_attachment" "ecs_ec2_role" {
  role       = aws_iam_role.container_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonEC2ContainerServiceforEC2Role"
}

resource "aws_iam_role_policy_attachment" "ssm_core" {
  role       = aws_iam_role.container_role.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_iam_role_policy_attachment" "ecs_task_execution_policy" {
  role       = aws_iam_role.container_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

resource "aws_iam_role_policy" "container_EC2_policy" {
  name   = "EC2DescribeInstances-policy"
  role   = aws_iam_role.container_role.name

  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Sid    = "VisualEditor0",
        Effect = "Allow",
        Action = "ec2:DescribeInstances",
        Resource = "*"
      }
    ]
  })
}

resource "aws_iam_role_policy" "container_secrets_policy" {
  name   = "Secret-manager-policy"
  role   = aws_iam_role.container_role.name

  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Sid    = "VisualEditor0",
        Effect = "Allow",
        Action = [
          "secretsmanager:GetSecretValue"
        ],
        Resource = [
          "arn:aws:secretsmanager:${var.aws_region}:${var.account_id}:secret:rerolldrinks/message/encryption/*",
          "arn:aws:secretsmanager:${var.aws_region}:${var.account_id}:secret:rerolldrinks/message/api-request-key/*"
        ]
      }
    ]
  })
}

resource "aws_iam_role_policy" "parameterstore_policy" {
  name   = "parameterstore-policy"
  role   = aws_iam_role.container_role.name

  policy = jsonencode({
    Version = "2012-10-17",
    Statement = [
      {
        Sid    = "VisualEditor0",
        Effect = "Allow",
        Action = [
          "ssm:GetParameterHistory",
          "ssm:GetParametersByPath",
          "ssm:GetParameters",
          "ssm:GetParameter"
        ],
        Resource = [
          "arn:aws:ssm:${var.aws_region}:${var.account_id}:parameter/rerolldrinks/development/signalr/conn",
          "arn:aws:ssm:${var.aws_region}:${var.account_id}:parameter/rerolldrinks/production/signalr/conn",
          "arn:aws:ssm:${var.aws_region}:${var.account_id}:parameter/rerolldrinks/development/sql/conn",
          "arn:aws:ssm:${var.aws_region}:${var.account_id}:parameter/rerolldrinks/production/sql/conn"
        ]
      },
      {
        Sid    = "VisualEditor1",
        Effect = "Allow",
        Action = [
          "ssm:DescribeParameters"
        ],
        Resource = [
          "*"
        ]
      }
    ]
  })
}

resource "aws_iam_instance_profile" "instance_profile" {
  name = "${var.service_name}"
  role = aws_iam_role.container_role.name
}

## -- Create Launch Template
resource "aws_launch_template" "launch_template" {
  name          = var.service_name
  image_id      = "ami-0f7301c9bf11d7375"
  instance_type = "t2.micro"
  key_name      = "rerolldrinks-kp"

  iam_instance_profile {
    name = aws_iam_instance_profile.instance_profile.name
  }

  user_data = base64encode(<<-EOF
    #!/bin/bash
    echo ECS_CLUSTER=${var.service_name} >> /etc/ecs/ecs.config;
  EOF
  )

  tag_specifications {
    resource_type = "instance"
    tags = {
      Name = "${var.service_name}"
    }
  }

  network_interfaces {
    subnet_id       = data.aws_subnet.subnet_a.id
    security_groups = ["${data.aws_security_group.reroll_drinks_default.id}"]
  }
}

## -- Create AutoScalingGroup
resource "aws_autoscaling_group" "auto_scaling_group" {
  name = var.service_name

  min_size           = 0
  max_size           = 1
  desired_capacity   = 0
  vpc_zone_identifier = [
    data.aws_subnet.subnet_a.id,
    data.aws_subnet.subnet_b.id,
    data.aws_subnet.subnet_c.id
  ]

  launch_template {
    id      = aws_launch_template.launch_template.id
    version = "$Latest"
  }

  health_check_type         = "EC2"
  health_check_grace_period = 300

  instance_maintenance_policy {
    min_healthy_percentage = 100
    max_healthy_percentage = 150
  }

  tag {
    key                 = "Name"
    value               = "MessageService"
    propagate_at_launch = true
  }
}

## -- Create Cluster
resource "aws_ecs_cluster" "ecs_cluster" {
  name = var.service_name
}

resource "aws_ecs_capacity_provider" "ec2_provider" {
  name = "${var.service_name}"
  auto_scaling_group_provider {
    auto_scaling_group_arn = aws_autoscaling_group.auto_scaling_group.arn
    managed_scaling {
      status                    = "ENABLED"
      target_capacity           = 100
      minimum_scaling_step_size = 1
      maximum_scaling_step_size = 100
    }
  }
  tags = {
    Name = "EC2_PROVIDER"
  }
}

# Attach the Capacity Provider to the ECS Cluster
resource "aws_ecs_cluster_capacity_providers" "cluster_capacity_providers" {
  cluster_name       = aws_ecs_cluster.ecs_cluster.id
  capacity_providers = [aws_ecs_capacity_provider.ec2_provider.name]

  default_capacity_provider_strategy {
    capacity_provider = aws_ecs_capacity_provider.ec2_provider.name
    weight            = 1
    base              = 1
  }
}

## -- create the ECR repo
resource "aws_ecr_repository" "ecr_repo" {
  name                 = "messageservice"
  image_tag_mutability = "MUTABLE"
}

## -- Create the cloudwatch loggroups
resource "aws_cloudwatch_log_group" "ecs_development_log_group" {
  name              = "/ecs/${var.service_name}-development"
  retention_in_days = 3
}

resource "aws_cloudwatch_log_group" "ecs_production_log_group" {
  name              = "/ecs/${var.service_name}-production"
  retention_in_days = 7
}

## -- Create Task Definitions
resource "aws_ecs_task_definition" "task_definition_development" {
  family                = "${var.service_name}-development"
  cpu                   = 256
  memory                = 512

  container_definitions = jsonencode([
    {
      name      = "${var.service_name}"
      image     = "${aws_ecr_repository.ecr_repo.name}:latest"
      cpu       = 256
      memory    = 512
      essential = true
      environment = [
        {
          name  = "ENV"
          value = "development"
        }
      ],
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = "/ecs/${var.service_name}-development"
          awslogs-region        = var.aws_region
          awslogs-stream-prefix = "development"
        }
      }
    }
  ])

  requires_compatibilities = ["EXTERNAL", "EC2"]
}

resource "aws_ecs_task_definition" "task_definition_production" {
  family       = "${var.service_name}-production"
  cpu          = 256
  memory       = 512

  container_definitions = jsonencode([
    {
      name      = "${var.service_name}"
      image     = "${aws_ecr_repository.ecr_repo.name}:latest"
      cpu       = 256
      memory    = 512
      essential = true
      environment = [
        {
          name  = "ENV"
          value = "production"
        }
      ],
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          awslogs-group         = "/ecs/${var.service_name}-production"
          awslogs-region        = var.aws_region
          awslogs-stream-prefix = "production"
        }
      }
    }
  ])

  requires_compatibilities = ["EXTERNAL", "EC2"]
}

## -- Create Services
resource "aws_ecs_service" "development" {
  name            = "${var.service_name}-development"
  cluster         = aws_ecs_cluster.ecs_cluster.id
  task_definition = aws_ecs_task_definition.task_definition_development.arn
  desired_count   = 0
  launch_type     = "EC2"
}

resource "aws_ecs_service" "production" {
  name            = "${var.service_name}-production"
  cluster         = aws_ecs_cluster.ecs_cluster.id
  task_definition = aws_ecs_task_definition.task_definition_production.arn
  desired_count   = 0
  launch_type     = "EC2"
}
