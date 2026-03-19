# Deployment Infrastructure

This folder contains all the infrastructure needed to deploy Go Modulith to the cloud.

## 📁 Structure

```
deployment/
├── helm/              # Helm charts for Kubernetes
│   └── modulith/      # Main chart (supports server and modules)
│       ├── values.yaml                  # Default values
│       ├── values-server.yaml           # Example: monolith deployment
│       ├── values-auth-module.yaml      # Example: auth module deployment
│       └── README.md                    # Complete documentation
├── opentofu/          # Base infrastructure modules
│   └── modules/
│       ├── vpc/       # Virtual network (subnets, NAT, IGW)
│       ├── eks/       # Kubernetes cluster
│       └── rds/       # PostgreSQL database
└── terragrunt/        # Environment-specific configuration
    └── envs/
        ├── dev/       # Development environment
        └── prod/      # Production environment
```

## 🚀 Complete Deployment Flow

### 1️⃣ Provision Base Infrastructure (IaC)

Use OpenTofu + Terragrunt to create VPC, EKS, and RDS:

```bash
cd deployment/terragrunt/envs/dev

# Create VPC
cd vpc && terragrunt apply

# Create EKS cluster
cd ../eks && terragrunt apply

# Create RDS PostgreSQL
cd ../rds && terragrunt apply
```

**Important outputs:**
- `vpc_id`, `subnet_ids`
- `eks_cluster_endpoint`, `eks_cluster_name`
- `rds_endpoint`, `rds_connection_string`

### 2️⃣ Configure kubectl

```bash
aws eks update-kubeconfig \
  --region us-east-1 \
  --name modulith-cluster-dev
```

### 3️⃣ Build and Push Docker Images

```bash
# Build images
just docker-build                # modulith-server:latest
just docker-build-module auth    # modulith-auth:latest

# Tag and push to registry (example: ECR)
docker tag modulith-server:latest 123456789.dkr.ecr.us-east-1.amazonaws.com/modulith-server:v1.0.0
docker push 123456789.dkr.ecr.us-east-1.amazonaws.com/modulith-server:v1.0.0
```

### 4️⃣ Deploy with Helm

#### Option A: Monolith

```bash
helm install modulith-server ./deployment/helm/modulith \
  --values ./deployment/helm/modulith/values-server.yaml \
  --set image.repository=123456789.dkr.ecr.us-east-1.amazonaws.com/modulith \
  --set image.tag=v1.0.0 \
  --set config.dbDsn="postgres://user:pass@rds-endpoint:5432/db" \
  --namespace production \
  --create-namespace
```

#### Option B: Independent Modules

```bash
# Deploy auth module
helm install modulith-auth ./deployment/helm/modulith \
  --values ./deployment/helm/modulith/values-auth-module.yaml \
  --set image.repository=123456789.dkr.ecr.us-east-1.amazonaws.com/modulith \
  --set image.tag=v1.0.0 \
  --namespace production
```

## 🔄 Migration Strategies

### Phase 1: Monolith (Start)

```
┌─────────────────────────────┐
│   EKS Cluster               │
│  ┌─────────────────────┐    │
│  │ modulith-server     │    │
│  │ (all modules)       │    │
│  └─────────────────────┘    │
└─────────────────────────────┘
         ↓
    ┌─────────┐
    │   RDS   │
    └─────────┘
```

**Advantages:**
- ✅ Simple to operate
- ✅ Lower latency between modules
- ✅ Native ACID transactions

### Phase 2: Hybrid (Transition)

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

**Advantages:**
- ✅ Scale critical modules independently
- ✅ Reduce blast radius of failures
- ✅ Maintain simplicity for stable modules

### Phase 3: Microservices (Advanced)

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

**Advantages:**
- ✅ Maximum scaling flexibility
- ✅ Independent deployments per team
- ✅ Complete failure isolation

## 📊 Monitoring and Observability

The project includes integration with:

- **OpenTelemetry**: Distributed tracing
- **Prometheus**: Metrics exposed at `/metrics`
- **Health Checks**: `/healthz` (liveness), `/readyz` (readiness)

### Configure Prometheus in K8s

```bash
# Install Prometheus Operator
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace

# ServiceMonitor is created automatically if you have annotations
```

## 🔐 Secrets Management

### Development

Use direct values in `values.yaml` (dev only):

```yaml
config:
  dbDsn: "postgres://dev:dev@localhost:5432/dev"
  jwtSecret: "dev-secret"
```

### Production

Use External Secrets Operator or Sealed Secrets:

```bash
# Example with External Secrets (AWS Secrets Manager)
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

## 🧪 Testing in Kubernetes

```bash
# Port forward for local testing
kubectl port-forward svc/modulith-server 8080:8080 -n production

# Health checks
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz

# gRPC testing
grpcurl -plaintext localhost:9050 list
```

## 📚 Additional Resources

- [Helm Chart README](./helm/modulith/README.md) - Detailed chart documentation
- [MODULITH_ARCHITECTURE.md](../docs/MODULITH_ARCHITECTURE.md) - Complete architecture
- [Makefile Commands](../README.md) - Build and deploy commands

## 🆘 Troubleshooting

### Pods don't start

```bash
kubectl describe pod <pod-name> -n production
kubectl logs <pod-name> -n production --previous
```

### RDS connectivity issues

Verify security groups and that pods are in the correct subnets:

```bash
# From a debug pod
kubectl run -it --rm debug --image=postgres:alpine --restart=Never -- sh
# Inside: psql "postgres://user:pass@rds-endpoint:5432/db"
```

### HPA doesn't scale

```bash
# Verify metrics-server
kubectl top nodes
kubectl top pods -n production

# Verify HPA status
kubectl describe hpa modulith-server -n production
```
