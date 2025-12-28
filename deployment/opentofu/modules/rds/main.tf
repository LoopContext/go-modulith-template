resource "aws_db_subnet_group" "main" {
  name       = "${var.project_name}-db-subnet-group-${var.environment}"
  subnet_ids = var.private_subnet_ids

  tags = {
    Name = "${var.project_name}-db-subnet-group-${var.environment}"
  }
}

resource "aws_security_group" "rds" {
  name        = "${var.project_name}-rds-sg-${var.environment}"
  description = "Security group for RDS"
  vpc_id      = var.vpc_id

  ingress {
    from_port   = 5432
    to_port     = 5432
    protocol    = "tcp"
    cidr_blocks = ["10.0.0.0/16"] # Allow within VPC
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "${var.project_name}-rds-sg-${var.environment}"
  }
}

resource "aws_db_instance" "main" {
  identifier           = "${var.project_name}-db-${var.environment}"
  allocated_storage    = 20
  db_name              = var.db_name
  engine               = "postgres"
  engine_version       = "16"
  instance_class       = var.instance_class
  username             = var.db_user
  password             = var.db_password
  db_subnet_group_name = aws_db_subnet_group.main.name
  vpc_security_group_ids = [aws_security_group.rds.id]
  skip_final_snapshot  = true # Only for dev/template purposes

  tags = {
    Name        = "${var.project_name}-db-${var.environment}"
    Environment = var.environment
  }
}

output "db_endpoint" {
  value = aws_db_instance.main.endpoint
}
