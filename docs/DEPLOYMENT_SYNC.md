# 🔄 Deployment Infrastructure Sync Report

Este documento resume la sincronización completa entre el código, build system, y la infraestructura de deployment.

## ✅ Estado de Sincronización

**Fecha:** Enero 2026
**Estado:** ✅ Totalmente Sincronizado

---

## 🧪 Testing y Calidad

### Mocking con gomock

El proyecto utiliza **gomock** para generación automática de mocks:

```bash
# Generar mocks de todas las interfaces
make generate-mocks

# Ejecutar tests unitarios (con mocks, sin DB)
make test-unit

# Tests completos (incluyendo integración)
make test
```

**Características:**
- ✅ Type-safe: Los mocks fallan en compilación si la interfaz cambia
- ✅ Automático: Regeneración mediante `go generate`
- ✅ Alineado: Misma filosofía que sqlc y buf

**Ver documentación:** `docs/MODULITH_ARCHITECTURE.md` (Sección: Mocking)

### Coverage Reporting

```bash
# Reporte visual en terminal con estadísticas
make coverage-report

# Reporte HTML interactivo
make coverage-html
```

**El reporte muestra:**
- 📦 Cobertura por paquete con indicadores visuales
- 📈 Estadísticas generales (excelente/buena/media)
- 🎯 Top 10 archivos con mejor cobertura
- ⚠️ Áreas que necesitan más tests

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

## 🔄 Build, Release, Run (12-Factor App: Factor V)

El template sigue el principio de **separación de build, release y run** de la metodología 12-factor app.

### Las Tres Etapas

**1. Build Stage:**
- Compila el código fuente en un ejecutable
- Genera código desde protobuf (buf)
- Genera código desde SQL (sqlc)
- Crea la imagen Docker
- **Resultado:** Artefacto ejecutable (binario o imagen)

**2. Release Stage:**
- Combina el build con la configuración del entorno
- Aplica migraciones de base de datos (opcional)
- Valida configuración
- **Resultado:** Release listo para ejecutar

**3. Run Stage:**
- Ejecuta la aplicación en el entorno objetivo
- Inicia los procesos (web, worker)
- **Resultado:** Aplicación en ejecución

### Implementación en el Template

#### Build Stage

```bash
# Build binario local
make build                    # → bin/server
make build-module auth        # → bin/auth

# Build imagen Docker
make docker-build             # → modulith-server:latest
make docker-build-module auth # → modulith-auth:latest
```

**Durante el build:**
- ✅ Generación de código (proto, sqlc)
- ✅ Compilación de binarios
- ✅ Creación de imagen Docker multi-stage
- ✅ Inclusión de version info (VERSION, COMMIT, BUILD_TIME)

#### Release Stage

**Opción 1: Migraciones en Startup (Recomendado para Modulith)**
```bash
# El servidor ejecuta migraciones automáticamente al iniciar
./bin/server  # Ejecuta migraciones, luego inicia servidor
```

**Ventajas:**
- ✅ Simple y directo
- ✅ Asegura que las migraciones se ejecuten
- ✅ Funciona bien para modulith (un solo proceso)

**Opción 2: Migraciones como Job Separado (Producción)**
```bash
# Ejecutar migraciones como job de Kubernetes
kubectl apply -f deployment/helm/modulith/templates/migration-job.yaml

# Luego iniciar la aplicación
helm install modulith-server ./deployment/helm/modulith
```

**Ventajas:**
- ✅ Separación clara de build/release/run
- ✅ Migraciones ejecutadas antes del deploy
- ✅ Rollback más seguro

**Recomendación:**
- **Desarrollo/Staging:** Migraciones en startup (Opción 1)
- **Producción:** Migraciones como job separado (Opción 2)

#### Run Stage

```bash
# Ejecutar aplicación
./bin/server

# O con Docker
docker run modulith-server:latest

# O en Kubernetes
helm install modulith-server ./deployment/helm/modulith
```

**Durante el run:**
- ✅ Carga configuración (YAML > .env > ENV vars)
- ✅ Conecta a servicios externos (DB, Redis)
- ✅ Ejecuta migraciones (si no se ejecutaron en release)
- ✅ Inicia servidores HTTP/gRPC
- ✅ Listo para recibir requests

### Separación de Responsabilidades

**Build:**
- ✅ Compilación de código
- ✅ Generación de artefactos
- ✅ Creación de imágenes
- ❌ NO ejecuta migraciones
- ❌ NO accede a base de datos
- ❌ NO requiere configuración de entorno

**Release:**
- ✅ Aplicación de configuración
- ✅ Ejecución de migraciones (opcional)
- ✅ Validación de configuración
- ❌ NO ejecuta la aplicación

**Run:**
- ✅ Ejecución de procesos
- ✅ Manejo de requests
- ✅ Gestión de ciclo de vida
- ❌ NO compila código
- ❌ NO aplica migraciones (si se hicieron en release)

### Ejemplo Completo: CI/CD Pipeline

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
          # Ejecutar migraciones como job separado
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

### Migraciones: Estrategia Híbrida

El template soporta ambas estrategias:

**1. Migraciones en Startup (Default):**
```go
// cmd/server/main.go
func main() {
    // ...
    runMigrations(cfg.DBDSN, reg)  // Ejecuta migraciones
    runServer(ctx, cfg, reg, stop)  // Inicia servidor
}
```

**2. Migraciones como Job (Kubernetes):**
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
        command: ["./service", "-migrate"]  # Solo migraciones
      restartPolicy: Never
```

**Uso:**
```bash
# Ejecutar migraciones antes del deploy
kubectl apply -f migration-job.yaml

# Esperar completación
kubectl wait --for=condition=complete job/migration-job

# Luego desplegar aplicación
helm install modulith-server ./deployment/helm/modulith
```

### Checklist de Build/Release/Run

**Build:**
- [ ] Código compilado sin errores
- [ ] Artefactos generados (proto, sqlc)
- [ ] Imagen Docker creada
- [ ] Version info incluida

**Release:**
- [ ] Configuración validada
- [ ] Migraciones ejecutadas (si aplica)
- [ ] Secrets configurados
- [ ] Health checks configurados

**Run:**
- [ ] Proceso inicia correctamente
- [ ] Conecta a servicios externos
- [ ] Health checks responden
- [ ] Logs estructurados funcionando

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

