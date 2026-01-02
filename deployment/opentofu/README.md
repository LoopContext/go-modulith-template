# OpenTofu Infrastructure Modules

Base infrastructure modules for deploying Go Modulith on AWS using OpenTofu (open-source fork of Terraform).

## 📁 Module Structure

```
opentofu/modules/
├── vpc/        # Virtual private network
├── eks/        # Kubernetes cluster
└── rds/        # PostgreSQL database
```

## 🏗️ Available Modules

### VPC Module

Creates a VPC with public and private subnets, NAT Gateway, and Internet Gateway.

**Resources created:**
- VPC with configurable CIDR
- Public subnets (for Load Balancers)
- Private subnets (for EKS nodes and RDS)
- Internet Gateway
- NAT Gateway
- Route tables

**Outputs:**
- `vpc_id`
- `public_subnet_ids`
- `private_subnet_ids`

### EKS Module

Creates a managed Kubernetes cluster with node groups.

**Resources created:**
- EKS Cluster
- IAM roles and policies
- Node groups with autoscaling
- Security groups

**Outputs:**
- `cluster_endpoint`
- `cluster_name`

### RDS Module

Creates a managed PostgreSQL instance.

**Resources created:**
- RDS PostgreSQL instance
- DB subnet group
- Security group (port 5432)

**Outputs:**
- `db_endpoint`
- `db_connection_string` (sensitive)

## 🚀 Usage with Terragrunt

Modules are used through Terragrunt to manage multiple environments:

```bash
cd deployment/terragrunt/envs/dev

# Initialize and apply VPC
cd vpc
terragrunt init
terragrunt plan
terragrunt apply

# Initialize and apply EKS (depends on VPC)
cd ../eks
terragrunt apply

# Initialize and apply RDS (depends on VPC)
cd ../rds
terragrunt apply
```

## 📋 Prerequisites

1. **OpenTofu installed:**
```bash
# macOS
brew install opentofu

# Linux
curl --proto '=https' --tlsv1.2 -fsSL https://get.opentofu.org/install-opentofu.sh | sh
```

2. **Terragrunt installed:**
```bash
# macOS
brew install terragrunt

# Linux
wget https://github.com/gruntwork-io/terragrunt/releases/download/v0.54.0/terragrunt_linux_amd64
chmod +x terragrunt_linux_amd64
sudo mv terragrunt_linux_amd64 /usr/local/bin/terragrunt
```

3. **AWS CLI configured:**
```bash
aws configure
# Enter: Access Key ID, Secret Access Key, Region
```

## 🔧 Environment Configuration

### Development (dev)

```hcl
# deployment/terragrunt/envs/dev/vpc/terragrunt.hcl
inputs = {
  environment = "dev"
  cidr_block  = "10.0.0.0/16"
}
```

### Production (prod)

```hcl
# deployment/terragrunt/envs/prod/vpc/terragrunt.hcl
inputs = {
  environment = "prod"
  cidr_block  = "10.1.0.0/16"
}
```

## 🔄 Complete Workflow

### 1. Provision Infrastructure

```bash
# From project root
cd deployment/terragrunt/envs/dev

# Apply all modules in order
terragrunt run-all apply
```

### 2. Get Outputs

```bash
# EKS cluster endpoint
cd eks && terragrunt output cluster_endpoint

# RDS connection string
cd ../rds && terragrunt output db_connection_string
```

### 3. Configure kubectl

```bash
aws eks update-kubeconfig \
  --region us-east-1 \
  --name $(cd eks && terragrunt output -raw cluster_name)
```

### 4. Verify Connectivity

```bash
# Verify nodes
kubectl get nodes

# Verify RDS is accessible from cluster
kubectl run -it --rm debug --image=postgres:alpine --restart=Never -- \
  psql "$(cd rds && terragrunt output -raw db_connection_string)"
```

## 🔐 Secrets Management

**⚠️ IMPORTANT:** Sensitive values (passwords, secrets) should NOT be in code.

### Option 1: Environment Variables

```bash
export TF_VAR_db_password="super-secret-password"
terragrunt apply
```

### Option 2: AWS Secrets Manager

```hcl
data "aws_secretsmanager_secret_version" "db_password" {
  secret_id = "prod/modulith/db-password"
}

resource "aws_db_instance" "main" {
  password = data.aws_secretsmanager_secret_version.db_password.secret_string
}
```

### Option 3: Terragrunt Inputs

```hcl
# terragrunt.hcl
inputs = {
  db_password = get_env("DB_PASSWORD", "default-only-for-dev")
}
```

## 🧹 Cleanup

To destroy all infrastructure:

```bash
cd deployment/terragrunt/envs/dev

# Destroy in reverse order
cd rds && terragrunt destroy
cd ../eks && terragrunt destroy
cd ../vpc && terragrunt destroy

# Or all at once (with caution!)
terragrunt run-all destroy
```

## 📊 Estimated Costs (AWS)

### Development Environment

| Resource | Type | Monthly Cost (approx.) |
|----------|------|------------------------|
| EKS Cluster | - | $73 |
| EC2 Nodes | 2x t3.medium | $60 |
| RDS | db.t3.micro | $15 |
| NAT Gateway | - | $32 |
| **Total** | | **~$180/month** |

### Production Environment

| Resource | Type | Monthly Cost (approx.) |
|----------|------|------------------------|
| EKS Cluster | - | $73 |
| EC2 Nodes | 3x t3.large | $190 |
| RDS | db.t3.medium | $60 |
| NAT Gateway | - | $32 |
| **Total** | | **~$355/month** |

**Note:** Approximate costs in us-east-1. May vary by region and usage.

## 🛡️ Best Practices

1. **Remote State**: Use S3 + DynamoDB for Terragrunt state
2. **Versioned Modules**: Use specific tags instead of `latest`
3. **Separate Environments**: Never share VPCs between dev and prod
4. **Backups**: Enable automatic backups in RDS for production
5. **Monitoring**: Configure CloudWatch alarms for critical resources
6. **Tagging**: Use consistent tags for cost allocation

## 🔍 Troubleshooting

### Error: "Error creating EKS Cluster"

Verify that:
- Subnets have the tags required by EKS
- IAM role has correct permissions
- You have sufficient quota in AWS

### Error: "Error creating RDS instance"

Verify that:
- Private subnets exist
- Security group allows traffic from EKS
- Password meets complexity requirements

### State Lock

If state is locked:

```bash
# Force unlock (with caution!)
terragrunt force-unlock <lock-id>
```

## 📚 Additional Resources

- [OpenTofu Documentation](https://opentofu.org/docs/)
- [Terragrunt Documentation](https://terragrunt.gruntwork.io/)
- [AWS EKS Best Practices](https://aws.github.io/aws-eks-best-practices/)
- [Deployment Guide](../README.md)
