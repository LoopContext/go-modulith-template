# 🔄 Deployment Infrastructure Sync Report

Este documento resume la sincronización completa entre el código, build system, y la infraestructura de deployment.

## ✅ Estado de Sincronización

**Fecha:** Diciembre 2025
**Estado:** ✅ Totalmente Sincronizado

---

## 📦 Build System

### Convención de Nombres de Binarios

Todos los binarios se compilan en `/bin/`:

| Comando | Output | Imagen Docker |
|---------|--------|---------------|
| `make build` | `bin/server` | `modulith-server:latest` |
| `make build-module auth` | `bin/auth` | `modulith-auth:latest` |
| `make build-module payments` | `bin/payments` | `modulith-payments:latest` |
| `make build-all` | `bin/*` | - |

### Docker Build

| Comando | Dockerfile ARG | Imagen Resultante |
|---------|----------------|-------------------|
| `make docker-build` | `TARGET=server` | `modulith-server:latest` |
| `make docker-build-module auth` | `TARGET=auth` | `modulith-auth:latest` |
| `make docker-build-module {module}` | `TARGET={module}` | `modulith-{module}:latest` |

**Dockerfile Path:** `/app/bin/service` (interno)

---

## ⚙️ Helm Charts

### Configuración Dinámica

El Helm chart soporta dos modos de deployment:

#### Modo 1: Server (Monolito)

```yaml
# values-server.yaml
deploymentType: server
# Genera imagen: modulith-server:latest
```

#### Modo 2: Module (Microservicio)

```yaml
# values-auth-module.yaml
deploymentType: module
moduleName: auth
# Genera imagen: modulith-auth:latest
```

### Archivos de Valores

| Archivo | Propósito | Deployment Type |
|---------|-----------|-----------------|
| `values.yaml` | Valores por defecto | `server` |
| `values-server.yaml` | Ejemplo monolito | `server` |
| `values-auth-module.yaml` | Ejemplo módulo auth | `module` |

### Puertos Configurados

| Servicio | HTTP | gRPC |
|----------|------|------|
| Server | 8080 | 9050 |
| Módulos | 8000 | 9000 |

**Health Checks:**
- Liveness: `/healthz`
- Readiness: `/readyz`

---

## 🏗️ Infraestructura (OpenTofu)

### Módulos Disponibles

```
deployment/opentofu/modules/
├── vpc/     → VPC, Subnets, NAT, IGW
├── eks/     → Kubernetes Cluster + Node Groups
└── rds/     → PostgreSQL Database
```

### Outputs Importantes

| Módulo | Output | Uso |
|--------|--------|-----|
| VPC | `vpc_id`, `subnet_ids` | Referencia para EKS/RDS |
| EKS | `cluster_endpoint`, `cluster_name` | kubectl config |
| RDS | `db_endpoint`, `db_connection_string` | App config |

### Gestión con Terragrunt

```
deployment/terragrunt/envs/
├── dev/      → Ambiente de desarrollo
│   ├── vpc/
│   ├── eks/
│   └── rds/
└── prod/     → Ambiente de producción
    ├── vpc/
    ├── eks/
    └── rds/
```

---

## 🔄 Flujo de Deployment Completo

### 1. Build Local

```bash
# Opción A: Binario local
make build-module auth
./bin/auth

# Opción B: Docker local
make docker-build-module auth
docker run modulith-auth:latest
```

### 2. Provisionar Infraestructura

```bash
cd deployment/terragrunt/envs/dev
terragrunt run-all apply
# Crea: VPC → EKS → RDS
```

### 3. Push a Registry

```bash
# Tag
docker tag modulith-auth:latest \
  123456789.dkr.ecr.us-east-1.amazonaws.com/modulith-auth:v1.0.0

# Push
docker push 123456789.dkr.ecr.us-east-1.amazonaws.com/modulith-auth:v1.0.0
```

### 4. Deploy con Helm

```bash
# Configurar kubectl
aws eks update-kubeconfig --name modulith-cluster-dev

# Deploy módulo
helm install modulith-auth ./deployment/helm/modulith \
  --values ./deployment/helm/modulith/values-auth-module.yaml \
  --set image.repository=123456789.dkr.ecr.us-east-1.amazonaws.com/modulith \
  --set image.tag=v1.0.0 \
  --namespace production
```

---

## 📊 Estrategias de Deployment

### Fase 1: Monolito

```
┌─────────────────────┐
│  make docker-build  │
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

### Fase 2: Híbrida

```
┌──────────────────┐  ┌──────────────────────────┐
│ make docker-build│  │make docker-build-module  │
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

### Fase 3: Microservicios

```
┌─────────────────────────────────────┐
│  make docker-build-module {module}  │
└──────────────┬──────────────────────┘
               ↓
┌──────────────────────────────────────┐
│  modulith-{module}:tag (cada uno)    │
└──────────────┬───────────────────────┘
               ↓
┌──────────────────────────────────────┐
│  Helm install por módulo             │
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

## 🔐 Configuración de Secretos

### Desarrollo

```yaml
# values.yaml
config:
  dbDsn: "postgres://dev:dev@localhost:5432/dev"
  jwtSecret: "dev-secret"
```

### Producción

```bash
# Desde Terragrunt outputs
DB_DSN=$(cd deployment/terragrunt/envs/prod/rds && \
  terragrunt output -raw db_connection_string)

# Deploy con secret
helm install modulith-server ./deployment/helm/modulith \
  --set config.dbDsn="${DB_DSN}" \
  --set config.jwtSecret="${JWT_SECRET}"
```

**Recomendado:** Usar External Secrets Operator o Sealed Secrets en producción.

---

## 📁 Estructura de Archivos Sincronizada

```
go-modulith-template/
├── bin/                          # Build outputs (gitignored)
│   ├── server                    # make build
│   ├── auth                      # make build-module auth
│   └── {module}                  # make build-module {module}
│
├── cmd/                          # Entry points
│   ├── server/main.go            # Monolito
│   └── {module}/main.go          # Módulos independientes
│
├── Dockerfile                    # Multi-stage, ARG TARGET
│
├── deployment/
│   ├── README.md                 # ✅ Guía completa
│   │
│   ├── helm/modulith/
│   │   ├── README.md             # ✅ Documentación Helm
│   │   ├── values.yaml           # ✅ Defaults
│   │   ├── values-server.yaml    # ✅ Ejemplo monolito
│   │   ├── values-auth-module.yaml # ✅ Ejemplo módulo
│   │   └── templates/
│   │       ├── deployment.yaml   # ✅ Soporta server/module
│   │       ├── service.yaml
│   │       ├── hpa.yaml
│   │       ├── pdb.yaml
│   │       └── secrets.yaml
│   │
│   ├── opentofu/
│   │   ├── README.md             # ✅ Documentación IaC
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
│   ├── MODULITH_ARCHITECTURE.md  # ✅ Actualizado sección K8s/IaC
│   └── DEPLOYMENT_SYNC.md        # ✅ Este documento
│
├── Makefile                      # ✅ Comandos genéricos
└── README.md                     # ✅ Referencias actualizadas
```

---

## ✅ Checklist de Sincronización

### Build System
- [x] Todos los binarios en `/bin/`
- [x] `.gitignore` actualizado
- [x] Comandos genéricos: `build-module`, `docker-build-module`
- [x] Convención de nombres: `modulith-{module}:tag`

### Helm Charts
- [x] Soporte para `deploymentType: server|module`
- [x] Nombres de imagen dinámicos
- [x] Valores de ejemplo para ambos modos
- [x] Health checks configurados
- [x] HPA y PDB incluidos
- [x] README completo con ejemplos

### OpenTofu/Terragrunt
- [x] Módulos VPC, EKS, RDS funcionales
- [x] Outputs necesarios definidos
- [x] README con guía de uso
- [x] Estructura por ambientes (dev/prod)

### Documentación
- [x] README principal actualizado
- [x] MODULITH_ARCHITECTURE.md con sección K8s/IaC
- [x] deployment/README.md con flujo completo
- [x] helm/modulith/README.md detallado
- [x] opentofu/README.md con ejemplos
- [x] Este documento de sincronización

---

## 🎯 Próximos Pasos Recomendados

### Para Desarrollo
1. ✅ Todo listo - usa `make dev-module {module}`

### Para Staging/Producción
1. Configurar AWS credentials
2. Provisionar infraestructura con Terragrunt
3. Configurar CI/CD para build y push de imágenes
4. Implementar External Secrets Operator
5. Configurar Prometheus + Grafana para observabilidad
6. Implementar GitOps con ArgoCD o Flux

---

## 📚 Referencias Rápidas

| Necesito... | Ver... |
|-------------|--------|
| Comandos de build | [README.md](../README.md) |
| Arquitectura completa | [MODULITH_ARCHITECTURE.md](./MODULITH_ARCHITECTURE.md) |
| Deployment en K8s | [deployment/README.md](../deployment/README.md) |
| Helm charts | [deployment/helm/modulith/README.md](../deployment/helm/modulith/README.md) |
| Infraestructura IaC | [deployment/opentofu/README.md](../deployment/opentofu/README.md) |

---

**Última actualización:** Diciembre 2025
**Mantenido por:** Go Modulith Template Team

