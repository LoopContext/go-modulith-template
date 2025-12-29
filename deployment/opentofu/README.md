# OpenTofu Infrastructure Modules

Módulos de infraestructura base para desplegar el Go Modulith en AWS usando OpenTofu (fork open-source de Terraform).

## 📁 Estructura de Módulos

```
opentofu/modules/
├── vpc/        # Red virtual privada
├── eks/        # Kubernetes cluster
└── rds/        # Base de datos PostgreSQL
```

## 🏗️ Módulos Disponibles

### VPC Module

Crea una VPC con subnets públicas y privadas, NAT Gateway e Internet Gateway.

**Recursos creados:**
- VPC con CIDR configurable
- Subnets públicas (para Load Balancers)
- Subnets privadas (para EKS nodes y RDS)
- Internet Gateway
- NAT Gateway
- Route tables

**Outputs:**
- `vpc_id`
- `public_subnet_ids`
- `private_subnet_ids`

### EKS Module

Crea un cluster de Kubernetes gestionado con node groups.

**Recursos creados:**
- EKS Cluster
- IAM roles y policies
- Node groups con autoscaling
- Security groups

**Outputs:**
- `cluster_endpoint`
- `cluster_name`

### RDS Module

Crea una instancia de PostgreSQL gestionada.

**Recursos creados:**
- RDS PostgreSQL instance
- DB subnet group
- Security group (puerto 5432)

**Outputs:**
- `db_endpoint`
- `db_connection_string` (sensitive)

## 🚀 Uso con Terragrunt

Los módulos se usan a través de Terragrunt para gestionar múltiples ambientes:

```bash
cd deployment/terragrunt/envs/dev

# Inicializar y aplicar VPC
cd vpc
terragrunt init
terragrunt plan
terragrunt apply

# Inicializar y aplicar EKS (depende de VPC)
cd ../eks
terragrunt apply

# Inicializar y aplicar RDS (depende de VPC)
cd ../rds
terragrunt apply
```

## 📋 Prerrequisitos

1. **OpenTofu instalado:**
```bash
# macOS
brew install opentofu

# Linux
curl --proto '=https' --tlsv1.2 -fsSL https://get.opentofu.org/install-opentofu.sh | sh
```

2. **Terragrunt instalado:**
```bash
# macOS
brew install terragrunt

# Linux
wget https://github.com/gruntwork-io/terragrunt/releases/download/v0.54.0/terragrunt_linux_amd64
chmod +x terragrunt_linux_amd64
sudo mv terragrunt_linux_amd64 /usr/local/bin/terragrunt
```

3. **AWS CLI configurado:**
```bash
aws configure
# Ingresa: Access Key ID, Secret Access Key, Region
```

## 🔧 Configuración por Ambiente

### Desarrollo (dev)

```hcl
# deployment/terragrunt/envs/dev/vpc/terragrunt.hcl
inputs = {
  environment = "dev"
  cidr_block  = "10.0.0.0/16"
}
```

### Producción (prod)

```hcl
# deployment/terragrunt/envs/prod/vpc/terragrunt.hcl
inputs = {
  environment = "prod"
  cidr_block  = "10.1.0.0/16"
}
```

## 🔄 Workflow Completo

### 1. Provisionar Infraestructura

```bash
# Desde la raíz del proyecto
cd deployment/terragrunt/envs/dev

# Aplicar todos los módulos en orden
terragrunt run-all apply
```

### 2. Obtener Outputs

```bash
# Endpoint del cluster EKS
cd eks && terragrunt output cluster_endpoint

# Connection string de RDS
cd ../rds && terragrunt output db_connection_string
```

### 3. Configurar kubectl

```bash
aws eks update-kubeconfig \
  --region us-east-1 \
  --name $(cd eks && terragrunt output -raw cluster_name)
```

### 4. Verificar Conectividad

```bash
# Verificar nodes
kubectl get nodes

# Verificar que RDS es accesible desde el cluster
kubectl run -it --rm debug --image=postgres:alpine --restart=Never -- \
  psql "$(cd rds && terragrunt output -raw db_connection_string)"
```

## 🔐 Gestión de Secretos

**⚠️ IMPORTANTE:** Los valores sensibles (passwords, secrets) NO deben estar en el código.

### Opción 1: Variables de Entorno

```bash
export TF_VAR_db_password="super-secret-password"
terragrunt apply
```

### Opción 2: AWS Secrets Manager

```hcl
data "aws_secretsmanager_secret_version" "db_password" {
  secret_id = "prod/modulith/db-password"
}

resource "aws_db_instance" "main" {
  password = data.aws_secretsmanager_secret_version.db_password.secret_string
}
```

### Opción 3: Terragrunt Inputs

```hcl
# terragrunt.hcl
inputs = {
  db_password = get_env("DB_PASSWORD", "default-only-for-dev")
}
```

## 🧹 Limpieza

Para destruir toda la infraestructura:

```bash
cd deployment/terragrunt/envs/dev

# Destruir en orden inverso
cd rds && terragrunt destroy
cd ../eks && terragrunt destroy
cd ../vpc && terragrunt destroy

# O todo a la vez (con cuidado!)
terragrunt run-all destroy
```

## 📊 Costos Estimados (AWS)

### Ambiente de Desarrollo

| Recurso | Tipo | Costo Mensual (aprox.) |
|---------|------|------------------------|
| EKS Cluster | - | $73 |
| EC2 Nodes | 2x t3.medium | $60 |
| RDS | db.t3.micro | $15 |
| NAT Gateway | - | $32 |
| **Total** | | **~$180/mes** |

### Ambiente de Producción

| Recurso | Tipo | Costo Mensual (aprox.) |
|---------|------|------------------------|
| EKS Cluster | - | $73 |
| EC2 Nodes | 3x t3.large | $190 |
| RDS | db.t3.medium | $60 |
| NAT Gateway | - | $32 |
| **Total** | | **~$355/mes** |

**Nota:** Costos aproximados en us-east-1. Pueden variar por región y uso.

## 🛡️ Mejores Prácticas

1. **State Remoto**: Usa S3 + DynamoDB para el state de Terragrunt
2. **Módulos Versionados**: Usa tags específicos en lugar de `latest`
3. **Ambientes Separados**: Nunca compartas VPCs entre dev y prod
4. **Backups**: Habilita backups automáticos en RDS para producción
5. **Monitoring**: Configura CloudWatch alarms para recursos críticos
6. **Tagging**: Usa tags consistentes para cost allocation

## 🔍 Troubleshooting

### Error: "Error creating EKS Cluster"

Verifica que:
- Las subnets tengan los tags requeridos por EKS
- El IAM role tenga los permisos correctos
- Tienes cuota suficiente en AWS

### Error: "Error creating RDS instance"

Verifica que:
- Las subnets privadas existen
- El security group permite tráfico desde EKS
- El password cumple los requisitos de complejidad

### State Lock

Si el state está bloqueado:

```bash
# Forzar unlock (con cuidado!)
terragrunt force-unlock <lock-id>
```

## 📚 Recursos Adicionales

- [OpenTofu Documentation](https://opentofu.org/docs/)
- [Terragrunt Documentation](https://terragrunt.gruntwork.io/)
- [AWS EKS Best Practices](https://aws.github.io/aws-eks-best-practices/)
- [Deployment Guide](../README.md)

