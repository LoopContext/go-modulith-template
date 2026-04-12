# 🔄 Deployment Infrastructure Sync Report

This document summarizes the complete synchronization between code, build system, and deployment infrastructure.

## ✅ Synchronization Status

**Date:** January 2026
**Status:** ✅ Fully Synchronized

---

## 🧪 Testing and Quality

### Mocking with gomock

The project uses **gomock** for automatic mock generation:

```bash
# Generate mocks for all interfaces
just generate-mocks

# Run unit tests (with mocks, no DB)
just test-unit

# Complete tests (including integration)
just test
```

**Features:**

-   ✅ Type-safe: Mocks fail at compilation if interface changes
-   ✅ Automatic: Regeneration via `go generate`
-   ✅ Aligned: Same philosophy as sqlc and buf

**See documentation:** `docs/MODULITH_ARCHITECTURE.md` (Section: Mocking)

### Coverage Reporting

```bash
# Visual report in terminal with statistics
just coverage-report

# Interactive HTML report
just coverage-html
```

**The report shows:**

-   📦 Coverage per package with visual indicators
-   📈 General statistics (excellent/good/medium)
-   🎯 Top 10 files with best coverage
-   ⚠️ Areas that need more tests

---

## 📦 Build System

### Binary Naming Convention

All binaries are compiled in `/bin/`:

| Command                      | Output         | Docker Image               |
| ---------------------------- | -------------- | -------------------------- |
| `just build`                 | `bin/server`   | `modulith-server:latest`   |
| `just build-module auth`     | `bin/auth`     | `modulith-auth:latest`     |
| `just build-module payments` | `bin/payments` | `modulith-payments:latest` |
| `just build-all`             | `bin/*`        | -                          |

### Docker Build

| Command                             | Dockerfile ARG    | Resulting Image            |
| ----------------------------------- | ----------------- | -------------------------- |
| `just docker-build`                 | `TARGET=server`   | `modulith-server:latest`   |
| `just docker-build-module auth`     | `TARGET=auth`     | `modulith-auth:latest`     |
| `just docker-build-module {module}` | `TARGET={module}` | `modulith-{module}:latest` |

**Dockerfile Path:** `/app/bin/service` (internal)

---

## ⚙️ Helm Charts

### Dynamic Configuration

The Helm chart supports two deployment modes:

#### Mode 1: Server (Monolith)

```yaml
# values-server.yaml
deploymentType: server
# Generates image: modulith-server:latest
```

#### Mode 2: Module (Microservice)

```yaml
# values-auth-module.yaml
deploymentType: module
moduleName: auth
# Generates image: modulith-auth:latest
```

### Value Files

| File                      | Purpose             | Deployment Type |
| ------------------------- | ------------------- | --------------- |
| `values.yaml`             | Default values      | `server`        |
| `values-server.yaml`      | Monolith example    | `server`        |
| `values-auth-module.yaml` | Auth module example | `module`        |

### Configured Ports

| Service | HTTP | gRPC |
| ------- | ---- | ---- |
| Server  | 8080 | 9050 |
| Modules | 8000 | 9000 |

**Health Checks:**

-   Liveness: `/healthz`
-   Readiness: `/readyz`

---

## 🏗️ Infrastructure (OpenTofu)

### Available Modules

```
deployment/opentofu/modules/
├── vpc/     → VPC, Subnets, NAT, IGW
├── eks/     → Kubernetes Cluster + Node Groups
└── rds/     → PostgreSQL Database
```

### Important Outputs

| Module | Output                                | Usage                 |
| ------ | ------------------------------------- | --------------------- |
| VPC    | `vpc_id`, `subnet_ids`                | Reference for EKS/RDS |
| EKS    | `cluster_endpoint`, `cluster_name`    | kubectl config        |
| RDS    | `db_endpoint`, `db_connection_string` | App config            |

### Management with Terragrunt

```
deployment/terragrunt/envs/
├── dev/      → Development environment
│   ├── vpc/
│   ├── eks/
│   └── rds/
└── prod/     → Production environment
    ├── vpc/
    ├── eks/
    └── rds/
```

---

## 🔄 Build, Release, Run (12-Factor App: Factor V)

The template follows the **separation of build, release and run** principle from the 12-factor app methodology.

### The Three Stages

**1. Build Stage:**

-   Compiles source code into an executable
-   Generates code from protobuf (buf)
-   Generates code from SQL (sqlc)
-   Creates Docker image
-   **Result:** Executable artifact (binary or image)

**2. Release Stage:**

-   Combines build with environment configuration
-   Applies database migrations (optional)
-   Validates configuration
-   **Result:** Release ready to execute

**3. Run Stage:**

-   Executes the application in the target environment
-   Starts processes (web, worker)
-   **Result:** Running application

### Implementation in the Template

#### Build Stage

```bash
# Local binary build
just build                    # → bin/server
just build-module auth        # → bin/auth

# Docker image build
just docker-build             # → modulith-server:latest
just docker-build-module auth # → modulith-auth:latest
```

**During build:**

-   ✅ Code generation (proto, sqlc)
-   ✅ Binary compilation
-   ✅ Multi-stage Docker image creation
-   ✅ Version info inclusion (VERSION, COMMIT, BUILD_TIME)

#### Release Stage

**Option 1: Migrations on Startup (Recommended for Modulith)**

```bash
# Server automatically runs migrations on startup
./bin/server  # Runs migrations, then starts server
```

**Advantages:**

-   ✅ Simple and direct
-   ✅ Ensures migrations run
-   ✅ Works well for modulith (single process)

**Option 2: Migrations as Separate Job (Production)**

```bash
# Run migrations as Kubernetes job
kubectl apply -f deployment/helm/modulith/templates/migration-job.yaml

# Then start application
helm install modulith-server ./deployment/helm/modulith
```

**Advantages:**

-   ✅ Clear separation of build/release/run
-   ✅ Migrations executed before deploy
-   ✅ Safer rollback

**Recommendation:**

-   **Development/Staging:** Migrations on startup (Option 1)
-   **Production:** Migrations as separate job (Option 2)

#### Run Stage

```bash
# Run application
./bin/server

# Or with Docker
docker run modulith-server:latest

# Or in Kubernetes
helm install modulith-server ./deployment/helm/modulith
```

**During run:**

-   ✅ Loads configuration (YAML > .env > ENV vars)
-   ✅ Connects to external services (DB, Valkey)
-   ✅ Executes migrations (if not executed in release)
-   ✅ Starts HTTP/gRPC servers
-   ✅ Ready to receive requests

### Responsibility Separation

**Build:**

-   ✅ Code compilation
-   ✅ Artifact generation
-   ✅ Image creation
-   ❌ Does NOT run migrations
-   ❌ Does NOT access database
-   ❌ Does NOT require environment configuration

**Release:**

-   ✅ Configuration application
-   ✅ Migration execution (optional)
-   ✅ Configuration validation
-   ❌ Does NOT run application

**Run:**

-   ✅ Process execution
-   ✅ Request handling
-   ✅ Lifecycle management
-   ❌ Does NOT compile code
-   ❌ Does NOT apply migrations (if done in release)

### Complete Example: CI/CD Pipeline

```yaml
# .github/workflows/deploy.yml
name: Deploy

on:
    push:
        branches: [main]

jobs:
    build:
        runs-on: ubuntu-latest
        steps:
            - uses: actions/checkout@v3
            - name: Build Docker image
              run: |
                  docker build \
                    --build-arg VERSION=${{ github.ref_name }} \
                    --build-arg COMMIT=${{ github.sha }} \
                    -t modulith-server:${{ github.sha }} .
            - name: Push to registry
              run: |
                  docker push modulith-server:${{ github.sha }}

    release:
        needs: build
        runs-on: ubuntu-latest
        steps:
            - name: Run migrations
              run: |
                  # Run migrations as separate job
                  kubectl create job migration-${{ github.sha }} \
                    --from=cronjob/migration-job \
                    --image=modulith-server:${{ github.sha }}

    deploy:
        needs: [build, release]
        runs-on: ubuntu-latest
        steps:
            - name: Deploy to Kubernetes
              run: |
                  helm upgrade --install modulith-server \
                    --set image.tag=${{ github.sha }} \
                    ./deployment/helm/modulith
```

### Migrations: Hybrid Strategy

The template supports both strategies:

**1. Migrations on Startup (Default):**

```go
// cmd/server/main.go
func main() {
    // ...
    // Migrations run automatically via migration.NewRunner(cfg.DBDSN, reg).RunAll()
    runServer(ctx, cfg, reg, stop)  // Starts server
}
```

**2. Migrations as Job (Kubernetes):**

```yaml
# deployment/helm/modulith/templates/migration-job.yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: {{ include "modulith.fullname" . }}-migration
spec:
  template:
    spec:
      containers:
      - name: migration
        image: "{{ .Values.image.repository }}-server:{{ .Values.image.tag }}"
        command: ["./service", "-migrate"]  # Migrations only
      restartPolicy: Never
```

**Usage:**

```bash
# Run migrations before deploy
kubectl apply -f migration-job.yaml

# Wait for completion
kubectl wait --for=condition=complete job/migration-job

# Then deploy application
helm install modulith-server ./deployment/helm/modulith
```

### Build/Release/Run Checklist

**Build:**

-   [ ] Code compiled without errors
-   [ ] Artifacts generated (proto, sqlc)
-   [ ] Docker image created
-   [ ] Version info included

**Release:**

-   [ ] Configuration validated
-   [ ] Migrations executed (if applicable)
-   [ ] Secrets configured
-   [ ] Health checks configured

**Run:**

-   [ ] Process starts correctly
-   [ ] Connects to external services
-   [ ] Health checks respond
-   [ ] Structured logs working

## 🔄 Complete Deployment Flow

### 1. Local Build

```bash
# Option A: Local binary
just build-module auth
./bin/auth

# Option B: Local Docker
just docker-build-module auth
docker run modulith-auth:latest
```

### 2. Provision Infrastructure

```bash
cd deployment/terragrunt/envs/dev
terragrunt run-all apply
# Creates: VPC → EKS → RDS
```

### 3. Push to Registry

```bash
# Tag
docker tag modulith-auth:latest \
  123456789.dkr.ecr.us-east-1.amazonaws.com/modulith-auth:v1.0.0

# Push
docker push 123456789.dkr.ecr.us-east-1.amazonaws.com/modulith-auth:v1.0.0
```

### 4. Deploy with Helm

```bash
# Configure kubectl
aws eks update-kubeconfig --name modulith-cluster-dev

# Deploy module
helm install modulith-auth ./deployment/helm/modulith \
  --values ./deployment/helm/modulith/values-auth-module.yaml \
  --set image.repository=123456789.dkr.ecr.us-east-1.amazonaws.com/modulith \
  --set image.tag=v1.0.0 \
  --namespace production
```

---

## 📊 Deployment Strategies

### Phase 1: Monolith

```
┌─────────────────────┐
│  just docker-build  │
└──────────┬──────────┘
           ↓
┌─────────────────────┐
│ modulith-server:tag │
└──────────┬──────────┘
           ↓
┌─────────────────────┐
│  Helm (server mode) │
└──────────┬──────────┘
           ↓
┌─────────────────────┐
│   EKS Deployment    │
│   (1 pod type)      │
└─────────────────────┘
```

### Phase 2: Hybrid

```
┌──────────────────┐  ┌──────────────────────────┐
│ just docker-build│  │just docker-build-module  │
│                  │  │         auth             │
└────────┬─────────┘  └────────┬─────────────────┘
         ↓                     ↓
┌────────────────┐    ┌────────────────┐
│modulith-server │    │ modulith-auth  │
└────────┬───────┘    └────────┬───────┘
         ↓                     ↓
┌────────────────┐    ┌────────────────┐
│ Helm (server)  │    │ Helm (module)  │
└────────┬───────┘    └────────┬───────┘
         ↓                     ↓
┌────────────────────────────────────┐
│         EKS Cluster                │
│  ┌──────────┐  ┌──────────┐       │
│  │  Server  │  │   Auth   │       │
│  │  Pod     │  │   Pod    │       │
│  └──────────┘  └──────────┘       │
└────────────────────────────────────┘
```

### Phase 3: Microservices

```
┌─────────────────────────────────────┐
│  just docker-build-module {module}  │
└──────────────┬──────────────────────┘
               ↓
┌──────────────────────────────────────┐
│  modulith-{module}:tag (each one)    │
└──────────────┬───────────────────────┘
               ↓
┌──────────────────────────────────────┐
│  Helm install per module             │
└──────────────┬───────────────────────┘
               ↓
┌──────────────────────────────────────┐
│         EKS Cluster                  │
│  ┌──────┐ ┌──────┐ ┌──────┐         │
│  │ Auth │ │Orders│ │ Pay..│         │
│  └──────┘ └──────┘ └──────┘         │
└──────────────────────────────────────┘
```

---

## 🔐 Secrets Configuration

### Development

```yaml
# values.yaml
config:
    dbDsn: "postgres://dev:dev@localhost:5432/dev"
    jwtSecret: "dev-secret"
```

### Production

```bash
# From Terragrunt outputs
DB_DSN=$(cd deployment/terragrunt/envs/prod/rds && \
  terragrunt output -raw db_connection_string)

# Deploy with secret
helm install modulith-server ./deployment/helm/modulith \
  --set config.dbDsn="${DB_DSN}" \
  --set config.jwtSecret="${JWT_SECRET}"
```

**Recommended:** Use External Secrets Operator or Sealed Secrets in production.

---

## 📁 Synchronized File Structure

```
go-modulith-template/
├── bin/                          # Build outputs (gitignored)
│   ├── server                    # just build
│   ├── auth                      # just build-module auth
│   └── {module}                  # just build-module {module}
│
├── cmd/                          # Entry points
│   ├── server/main.go            # Monolith
│   └── {module}/main.go          # Independent modules
│
├── Dockerfile                    # Multi-stage, ARG TARGET
│
├── deployment/
│   ├── README.md                 # ✅ Complete guide
│   │
│   ├── helm/modulith/
│   │   ├── README.md             # ✅ Helm documentation
│   │   ├── values.yaml           # ✅ Defaults
│   │   ├── values-server.yaml    # ✅ Monolith example
│   │   ├── values-auth-module.yaml # ✅ Module example
│   │   └── templates/
│   │       ├── deployment.yaml   # ✅ Supports server/module
│   │       ├── service.yaml
│   │       ├── hpa.yaml
│   │       ├── pdb.yaml
│   │       └── secrets.yaml
│   │
│   ├── opentofu/
│   │   ├── README.md             # ✅ IaC documentation
│   │   └── modules/
│   │       ├── vpc/
│   │       ├── eks/
│   │       └── rds/              # ✅ Output db_connection_string
│   │
│   └── terragrunt/
│       └── envs/
│           ├── dev/
│           └── prod/
│
├── docs/
│   ├── MODULITH_ARCHITECTURE.md  # ✅ Updated K8s/IaC section
│   └── DEPLOYMENT_SYNC.md        # ✅ This document
│
├── justfile                      # ✅ Generic commands
└── README.md                     # ✅ Updated references
```

---

## ✅ Synchronization Checklist

### Build System

-   [x] All binaries in `/bin/`
-   [x] `.gitignore` updated
-   [x] Generic commands: `build-module`, `docker-build-module`
-   [x] Naming convention: `modulith-{module}:tag`

### Helm Charts

-   [x] Support for `deploymentType: server|module`
-   [x] Dynamic image names
-   [x] Example values for both modes
-   [x] Health checks configured
-   [x] HPA and PDB included
-   [x] Complete README with examples

### OpenTofu/Terragrunt

-   [x] VPC, EKS, RDS modules functional
-   [x] Necessary outputs defined
-   [x] README with usage guide
-   [x] Structure by environments (dev/prod)

### Documentation

-   [x] Main README updated
-   [x] MODULITH_ARCHITECTURE.md with K8s/IaC section
-   [x] deployment/README.md with complete flow
-   [x] helm/modulith/README.md detailed
-   [x] opentofu/README.md with examples
-   [x] This synchronization document

---

## 🎯 Recommended Next Steps

### For Development

1. ✅ Everything ready - use `just dev-module {module}`

### For Staging/Production

1. Configure AWS credentials
2. Provision infrastructure with Terragrunt
3. Configure CI/CD for image build and push
4. Implement External Secrets Operator
5. Configure Prometheus + Grafana for observability
6. Implement GitOps with ArgoCD or Flux

---

## 📚 Quick References

| I need...             | See...                                                                      |
| --------------------- | --------------------------------------------------------------------------- |
| Build commands        | [README.md](../README.md)                                                   |
| Complete architecture | [MODULITH_ARCHITECTURE.md](./MODULITH_ARCHITECTURE.md)                      |
| K8s deployment        | [deployment/README.md](../deployment/README.md)                             |
| Helm charts           | [deployment/helm/modulith/README.md](../deployment/helm/modulith/README.md) |
| IaC infrastructure    | [deployment/opentofu/README.md](../deployment/opentofu/README.md)           |

---

**Last updated:** December 2025
**Maintained by:** Go Modulith Template Team
