# Contributing to Go Modulith Template

¡Gracias por tu interés en contribuir! Este documento proporciona pautas para contribuir al proyecto.

## 🚀 Inicio Rápido

1. **Fork el repositorio**
2. **Clona tu fork**:
   ```bash
   git clone https://github.com/TU_USUARIO/go-modulith-template.git
   cd go-modulith-template
   ```

3. **Instala dependencias**:
   ```bash
   make install-deps
   ```

4. **Levanta la infraestructura**:
   ```bash
   make docker-up
   ```

5. **Ejecuta los tests**:
   ```bash
   make test
   make lint
   ```

## 📋 Proceso de Contribución

### 1. Crea una rama para tu feature

```bash
git checkout -b feature/nombre-descriptivo
```

### 2. Realiza tus cambios

Asegúrate de seguir las convenciones del proyecto:

- **Código Go**: Sigue las [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- **Commits**: Usa mensajes descriptivos en español
- **Tests**: Agrega tests para nuevas funcionalidades
- **Documentación**: Actualiza la documentación relevante

### 3. Ejecuta validaciones

**OBLIGATORIO antes de hacer commit:**

```bash
# Linter (debe pasar sin errores)
make lint

# Tests
make test

# Coverage (opcional pero recomendado)
make coverage-report
```

### 4. Commit y Push

```bash
git add .
git commit -m "feat: descripción breve del cambio"
git push origin feature/nombre-descriptivo
```

### 5. Crea un Pull Request

- Usa un título descriptivo
- Explica qué cambia y por qué
- Referencia issues relacionados (si aplica)
- Asegúrate de que el CI pase

## 🔍 Guías de Estilo

### Código Go

- **Linting**: El proyecto usa `golangci-lint` con configuración estricta
- **Formato**: Todo el código debe pasar `gofmt` y `goimports`
- **Nombrado**: Sigue las convenciones estándar de Go
- **Errores**: Siempre wrappea errores con contexto usando `fmt.Errorf("context: %w", err)`

### Tests

- **Unit tests**: Para lógica de negocio (usa mocks de `gomock`)
- **Integration tests**: Para operaciones de DB (con flag `-short`)
- **Coverage mínimo**: Aspira a >60% en código nuevo

### Documentación

- **README**: Actualiza si agregas features visibles
- **Código**: Documenta funciones/tipos públicos con GoDoc
- **Docs**: Actualiza documentos en `/docs/` si es relevante

## 📝 Tipos de Commits

Usa prefijos semánticos:

- `feat:` - Nueva funcionalidad
- `fix:` - Corrección de bug
- `docs:` - Cambios en documentación
- `refactor:` - Refactorización sin cambio de comportamiento
- `test:` - Agregar o modificar tests
- `chore:` - Cambios en build, deps, etc.

### Actualizar CHANGELOG.md

Cuando agregues features, fixes, o cambios importantes, actualiza `CHANGELOG.md`:

1. Agrega tu cambio en la sección `[Unreleased]` apropiada
2. Usa las categorías: Added, Changed, Deprecated, Removed, Fixed, Security
3. Sigue el formato existente
4. Los cambios se moverán a una versión específica en el próximo release

## 🐛 Reportar Bugs

Cuando reportes un bug, incluye:

1. **Descripción**: Qué esperabas vs qué pasó
2. **Pasos para reproducir**
3. **Versión de Go**: `go version`
4. **Logs relevantes**: Si aplica

## 💡 Proponer Features

Para proponer nuevas funcionalidades:

1. **Abre un issue** primero para discutirlo
2. Explica el caso de uso
3. Considera el impacto en la arquitectura
4. Espera feedback antes de implementar

## ⚠️ Consideraciones Importantes

### No Modificar Sin Justificación

- `.golangci.yaml` - No suavizar reglas de linting
- `sqlc.yaml` - Solo cambiar para nuevos módulos
- `buf.yaml` - Configuración de protobuf estándar

### Arquitectura

Este es un template **modulith**:
- Mantén el aislamiento entre módulos
- Usa eventos para comunicación cross-module
- Sigue el patrón de registry para DI
- Documenta decisiones arquitectónicas importantes

### Performance

- No optimizar prematuramente
- Si agregas código crítico para performance, incluye benchmarks
- Usa `go test -bench=.` para validar

## 🤝 Código de Conducta

- Sé respetuoso y constructivo
- Acepta feedback con mente abierta
- Enfócate en el código, no en las personas
- Ayuda a otros contributors cuando puedas

## 📧 Contacto

Si tienes preguntas, abre un issue o discusión en GitHub.

---

¡Gracias por contribuir! 🚀

