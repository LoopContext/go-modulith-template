# Deployment Infrastructure

Esta carpeta contiene toda la infraestructura necesaria para desplegar el Go Modulith en la nube.

## 📁 Estructura

```
deployment/
├── helm/              # Helm charts para Kubernetes
│   └── modulith/      # Chart principal (soporta server y módulos)
│       ├── values.yaml                  # Valores por defecto
│       ├── values-server.yaml           # Ejemplo: deployment monolito
│       ├── values-auth-module.yaml      # Ejemplo: deployment módulo auth
│       └── README.md                    # Documentación completa
├── opentofu/          # Módulos de infraestructura base
│   └── modules/
│       ├── vpc/       # Red virtual (subnets, NAT, IGW)
│       ├── eks/       # Kubernetes cluster
│       └── rds/       # Base de datos PostgreSQL
└── terragrunt/        # Configuración por ambiente
    └── envs/
        ├── dev/       # Ambiente de desarrollo
        └── prod/      # Ambiente de producción
```

## 🚀 Flujo de Despliegue Completo

### 1️⃣ Provisionar Infraestructura Base (IaC)

Usa OpenTofu + Terragrunt para crear VPC, EKS y RDS:

```bash
cd deployment/terragrunt/envs/dev

# Crear VPC
cd vpc && terragrunt apply

# Crear EKS cluster
cd ../eks && terragrunt apply

# Crear RDS PostgreSQL
cd ../rds && terragrunt apply
```

**Salidas importantes:**
- `vpc_id`, `subnet_ids`
- `eks_cluster_endpoint`, `eks_cluster_name`
- `rds_endpoint`, `rds_connection_string`

### 2️⃣ Configurar kubectl

```bash
aws eks update-kubeconfig \
  --region us-east-1 \
  --name modulith-cluster-dev
```

### 3️⃣ Construir y Publicar Imágenes Docker

```bash
# Construir imágenes
make docker-build                # modulith-server:latest
make docker-build-module auth    # modulith-auth:latest

# Tag y push a registry (ejemplo: ECR)
docker tag modulith-server:latest 123456789.dkr.ecr.us-east-1.amazonaws.com/modulith-server:v1.0.0
docker push 123456789.dkr.ecr.us-east-1.amazonaws.com/modulith-server:v1.0.0
```

### 4️⃣ Desplegar con Helm

#### Opción A: Monolito

```bash
helm install modulith-server ./deployment/helm/modulith \
  --values ./deployment/helm/modulith/values-server.yaml \
  --set image.repository=123456789.dkr.ecr.us-east-1.amazonaws.com/modulith \
  --set image.tag=v1.0.0 \
  --set config.dbDsn="postgres://user:pass@rds-endpoint:5432/db" \
  --namespace production \
  --create-namespace
```

#### Opción B: Módulos Independientes

```bash
# Desplegar módulo auth
helm install modulith-auth ./deployment/helm/modulith \
  --values ./deployment/helm/modulith/values-auth-module.yaml \
  --set image.repository=123456789.dkr.ecr.us-east-1.amazonaws.com/modulith \
  --set image.tag=v1.0.0 \
  --namespace production
```

## 🔄 Estrategias de Migración

### Fase 1: Monolito (Inicio)

```
┌─────────────────────────────┐
│   EKS Cluster               │
│  ┌─────────────────────┐    │
│  │ modulith-server     │    │
│  │ (todos los módulos) │    │
│  └─────────────────────┘    │
└─────────────────────────────┘
         ↓
    ┌─────────┐
    │   RDS   │
    └─────────┘
```

**Ventajas:**
- ✅ Simple de operar
- ✅ Menor latencia entre módulos
- ✅ Transacciones ACID nativas

### Fase 2: Híbrida (Transición)

```
┌─────────────────────────────┐
│   EKS Cluster               │
│  ┌─────────────────────┐    │
│  │ modulith-server     │    │
│  │ (core modules)      │    │
│  └─────────────────────┘    │
│  ┌─────────────────────┐    │
│  │ modulith-auth       │    │
│  │ (HPA: 2-10)         │    │
│  └─────────────────────┘    │
└─────────────────────────────┘
         ↓
    ┌─────────┐
    │   RDS   │
    └─────────┘
```

**Ventajas:**
- ✅ Escala módulos críticos independientemente
- ✅ Reduce blast radius de fallos
- ✅ Mantiene simplicidad para módulos estables

### Fase 3: Microservicios (Avanzada)

```
┌─────────────────────────────┐
│   EKS Cluster               │
│  ┌─────────────────────┐    │
│  │ modulith-auth       │    │
│  └─────────────────────┘    │
│  ┌─────────────────────┐    │
│  │ modulith-orders     │    │
│  └─────────────────────┘    │
│  ┌─────────────────────┐    │
│  │ modulith-payments   │    │
│  └─────────────────────┘    │
└─────────────────────────────┘
         ↓
    ┌─────────┐
    │   RDS   │
    └─────────┘
```

**Ventajas:**
- ✅ Máxima flexibilidad de escalado
- ✅ Deployments independientes por equipo
- ✅ Aislamiento total de fallos

## 📊 Monitoreo y Observabilidad

El proyecto incluye integración con:

- **OpenTelemetry**: Tracing distribuido
- **Prometheus**: Métricas expuestas en `/metrics`
- **Health Checks**: `/healthz` (liveness), `/readyz` (readiness)

### Configurar Prometheus en K8s

```bash
# Instalar Prometheus Operator
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace

# ServiceMonitor se crea automáticamente si tienes annotations
```

## 🔐 Gestión de Secretos

### Desarrollo

Usa valores directos en `values.yaml` (solo para dev):

```yaml
config:
  dbDsn: "postgres://dev:dev@localhost:5432/dev"
  jwtSecret: "dev-secret"
```

### Producción

Usa External Secrets Operator o Sealed Secrets:

```bash
# Ejemplo con External Secrets (AWS Secrets Manager)
kubectl apply -f - <<EOF
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: modulith-secrets
spec:
  secretStoreRef:
    name: aws-secrets-manager
  target:
    name: modulith-server-secrets
  data:
    - secretKey: db-dsn
      remoteRef:
        key: prod/modulith/db-dsn
    - secretKey: jwt-secret
      remoteRef:
        key: prod/modulith/jwt-secret
EOF
```

## 🧪 Testing en Kubernetes

```bash
# Port forward para testing local
kubectl port-forward svc/modulith-server 8080:8080 -n production

# Health checks
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz

# gRPC testing
grpcurl -plaintext localhost:9050 list
```

## 📚 Recursos Adicionales

- [Helm Chart README](./helm/modulith/README.md) - Documentación detallada del chart
- [MODULITH_ARCHITECTURE.md](../docs/MODULITH_ARCHITECTURE.md) - Arquitectura completa
- [Makefile Commands](../README.md) - Comandos de build y deploy

## 🆘 Troubleshooting

### Pods no inician

```bash
kubectl describe pod <pod-name> -n production
kubectl logs <pod-name> -n production --previous
```

### Problemas de conectividad a RDS

Verifica security groups y que los pods estén en las subnets correctas:

```bash
# Desde un pod de debug
kubectl run -it --rm debug --image=postgres:alpine --restart=Never -- sh
# Dentro: psql "postgres://user:pass@rds-endpoint:5432/db"
```

### HPA no escala

```bash
# Verifica metrics-server
kubectl top nodes
kubectl top pods -n production

# Verifica HPA status
kubectl describe hpa modulith-server -n production
```

