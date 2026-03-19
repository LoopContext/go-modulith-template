# Modulith Helm Chart

Este Helm chart facilita el despliegue del Go Modulith tanto como monolito completo o como módulos independientes en Kubernetes.

## 🎯 Características

- ✅ Soporte para deployment del servidor monolito
- ✅ Soporte para deployments de módulos individuales
- ✅ Horizontal Pod Autoscaling (HPA)
- ✅ Pod Disruption Budgets (PDB)
- ✅ Health checks (liveness y readiness probes)
- ✅ Configuración mediante Secrets de Kubernetes
- ✅ Configuración de recursos y límites

## 📦 Instalación

### Opción 1: Desplegar el Monolito (Servidor Completo)

```bash
helm install modulith-server ./deployment/helm/modulith \
  --values ./deployment/helm/modulith/values-server.yaml \
  --namespace production \
  --create-namespace
```

### Opción 2: Desplegar un Módulo Individual

Para desplegar el módulo `auth` de forma independiente:

```bash
helm install modulith-auth ./deployment/helm/modulith \
  --values ./deployment/helm/modulith/values-auth-module.yaml \
  --namespace production \
  --create-namespace
```

Para otros módulos, crea un archivo de valores similar ajustando `moduleName`:

```yaml
deploymentType: module
moduleName: payments  # o cualquier otro módulo
```

## 🔧 Configuración

### Valores Principales

| Parámetro | Descripción | Default |
|-----------|-------------|---------|
| `deploymentType` | Tipo de deployment: `server` o `module` | `server` |
| `moduleName` | Nombre del módulo (solo si `deploymentType=module`) | `""` |
| `replicaCount` | Número de réplicas | `1` |
| `image.repository` | Repositorio base de la imagen | `modulith` |
| `image.tag` | Tag de la imagen | `latest` |
| `service.httpPort` | Puerto HTTP | `8080` |
| `service.grpcPort` | Puerto gRPC | `9050` |
| `autoscaling.enabled` | Habilitar HPA | `false` |
| `podDisruptionBudget.enabled` | Habilitar PDB | `false` |

### Convención de Nombres de Imágenes

El chart construye el nombre completo de la imagen automáticamente:

- **Servidor**: `{repository}-server:{tag}` → `modulith-server:latest`
- **Módulo**: `{repository}-{moduleName}:{tag}` → `modulith-auth:latest`

Esto se alinea con los comandos del Makefile:
```bash
just docker-build          # Crea modulith-server:latest
just docker-build-module auth  # Crea modulith-auth:latest
```

### Secretos

Los secretos se configuran en `config` del values.yaml:

```yaml
config:
  env: prod
  dbDsn: "postgres://user:pass@host:5432/db"
  jwtSecret: "tu-secreto-jwt"
```

**⚠️ Importante**: En producción, usa Sealed Secrets, External Secrets, o Vault en lugar de valores en texto plano.

## 🚀 Ejemplos de Uso

### Desarrollo (Minikube)

```bash
# Deploy del servidor
helm install modulith ./deployment/helm/modulith \
  --set image.tag=dev \
  --set config.env=dev

# Deploy del módulo auth
helm install modulith-auth ./deployment/helm/modulith \
  --set deploymentType=module \
  --set moduleName=auth \
  --set image.tag=dev \
  --set config.env=dev
```

### Producción

```bash
# Con archivo de valores personalizado
helm install modulith-prod ./deployment/helm/modulith \
  --values values-production.yaml \
  --namespace production
```

### Actualización

```bash
# Actualizar el deployment
helm upgrade modulith-server ./deployment/helm/modulith \
  --values values-server.yaml \
  --namespace production

# Cambiar versión de la imagen
helm upgrade modulith-server ./deployment/helm/modulith \
  --reuse-values \
  --set image.tag=v1.2.0 \
  --namespace production
```

## 🔍 Verificación

```bash
# Ver el estado del release
helm status modulith-server -n production

# Ver los pods
kubectl get pods -n production -l app.kubernetes.io/name=modulith

# Ver logs
kubectl logs -n production -l app.kubernetes.io/name=modulith -f

# Verificar health checks
kubectl port-forward -n production svc/modulith-server 8080:8080
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

## 🏗️ Arquitectura de Deployments

### Estrategia 1: Monolito (Fase Inicial)

```
┌─────────────────┐
│ modulith-server │  ← Todos los módulos en un pod
│  (HPA: 2-10)    │
└─────────────────┘
```

### Estrategia 2: Híbrida (Fase de Transición)

```
┌─────────────────┐
│ modulith-server │  ← Módulos core
└─────────────────┘
         +
┌─────────────────┐
│ modulith-auth   │  ← Módulo separado (alta demanda)
└─────────────────┘
```

### Estrategia 3: Microservicios (Fase Avanzada)

```
┌─────────────────┐
│ modulith-auth   │
└─────────────────┘
┌─────────────────┐
│ modulith-orders │
└─────────────────┘
┌─────────────────┐
│ modulith-pay... │
└─────────────────┘
```

## 📚 Recursos Adicionales

- [Documentación de Arquitectura](../../../docs/MODULITH_ARCHITECTURE.md)
- [Makefile Commands](../../../README.md#-comandos-útiles-makefile)
- [Dockerfile Multi-stage Build](../../../Dockerfile)

## 🛡️ Mejores Prácticas

1. **Usa PDB en producción** para evitar interrupciones durante actualizaciones
2. **Configura HPA** basado en métricas reales de carga
3. **Usa Secrets externos** (Vault, AWS Secrets Manager, etc.)
4. **Configura resource limits** apropiados para evitar OOMKilled
5. **Monitorea con Prometheus** las métricas expuestas por la aplicación
6. **Usa diferentes namespaces** por ambiente (dev, staging, prod)

## 🐛 Troubleshooting

### Pod no inicia

```bash
kubectl describe pod <pod-name> -n production
kubectl logs <pod-name> -n production
```

### Health checks fallan

Verifica que:
- La aplicación expone `/healthz` y `/readyz` en el puerto HTTP
- El puerto configurado coincide con la aplicación
- La base de datos es accesible desde el pod

### Problemas de conexión a DB

```bash
# Verifica el secret
kubectl get secret modulith-server-secrets -n production -o yaml

# Prueba conectividad desde el pod
kubectl exec -it <pod-name> -n production -- sh
# Dentro del pod: intenta conectar a la DB
```

