#!/bin/bash

# Script to generate a detailed coverage report
# Usage: ./scripts/coverage-report.sh

set -e

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$PROJECT_ROOT"

echo "🧪 Ejecutando tests y generando reporte de cobertura..."
echo ""

# Run tests with coverage
go test ./... -coverprofile=coverage.out -covermode=atomic > /dev/null 2>&1

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║         📊 REPORTE DE COBERTURA - Go Modulith Template        ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

echo "┌─────────────────────────────────────────────────────────────────┐"
echo "│  📦 Cobertura por Paquete                                       │"
echo "└─────────────────────────────────────────────────────────────────┘"
echo ""

# Get coverage by package (excluding generated code and 0%)
go test ./... -cover 2>&1 | \
  grep "coverage:" | \
  grep -v "0.0%" | \
  grep -v "no test" | \
  awk '{
    # Extract package name and coverage
    if ($2 ~ /github.com/) {
      pkg = $2
      gsub("github.com/cmelgarejo/go-modulith-template/", "", pkg)
      cov = $(NF-2)

      # Add emoji based on coverage
      if (cov >= "95.0%") emoji = "🟢"
      else if (cov >= "80.0%") emoji = "🟡"
      else if (cov >= "60.0%") emoji = "🟠"
      else emoji = "🔴"

      printf "  %s  %-50s %s\n", emoji, pkg, cov
    }
  }' | sort -t'%' -k2 -rn

echo ""
echo "┌─────────────────────────────────────────────────────────────────┐"
echo "│  📈 Estadísticas Generales                                      │"
echo "└─────────────────────────────────────────────────────────────────┘"
echo ""

# Calculate stats
TOTAL_LINES=$(go tool cover -func=coverage.out | tail -1 | awk '{print $3}')
TESTED_PACKAGES=$(go test ./... -cover 2>&1 | grep "coverage:" | grep -v "0.0%" | grep -v "no test" | wc -l | tr -d ' ')
TOTAL_PACKAGES=$(find . -name "*.go" -not -path "./gen/*" -not -path "./.git/*" -not -path "./vendor/*" | xargs -I {} dirname {} | sort -u | wc -l | tr -d ' ')

# Count lines excluding generated code
HANDWRITTEN_LINES=$(go tool cover -func=coverage.out | \
  grep -v "\.pb\.go" | \
  grep -v "\.pb\.gw\.go" | \
  grep -v "/gen/" | \
  grep -v "cmd/auth/main.go" | \
  grep -v "cmd/server/main.go" | \
  tail -1 | \
  awk '{print $3}')

echo "  📊 Cobertura Total:              $TOTAL_LINES"
echo "  🎯 Código Escrito (sin gen):     $HANDWRITTEN_LINES"
echo "  📦 Paquetes con Tests:           $TESTED_PACKAGES"
echo "  🟢 Cobertura Excelente (>95%):   $(go test ./... -cover 2>&1 | grep -E '9[5-9]\.[0-9]%|100\.0%' | wc -l | tr -d ' ')"
echo "  🟡 Cobertura Buena (80-95%):     $(go test ./... -cover 2>&1 | grep -E '8[0-9]\.[0-9]%|9[0-4]\.[0-9]%' | wc -l | tr -d ' ')"
echo "  🟠 Cobertura Media (60-80%):     $(go test ./... -cover 2>&1 | grep -E '6[0-9]\.[0-9]%|7[0-9]\.[0-9]%' | wc -l | tr -d ' ')"
echo "  🔴 Cobertura Baja (<60%):        $(go test ./... -cover 2>&1 | grep -E '[0-5][0-9]\.[0-9]%' | grep -v '0.0%' | wc -l | tr -d ' ')"
echo ""

echo "┌─────────────────────────────────────────────────────────────────┐"
echo "│  🎯 Top 10 Archivos con Mejor Cobertura                        │"
echo "└─────────────────────────────────────────────────────────────────┘"
echo ""

go tool cover -func=coverage.out | \
  grep -v "\.pb\.go" | \
  grep -v "\.pb\.gw\.go" | \
  grep -v "/gen/" | \
  grep -v "cmd/" | \
  grep -v "total:" | \
  sort -k3 -t' ' -rn | \
  head -10 | \
  awk -F':' '{
    gsub("github.com/cmelgarejo/go-modulith-template/", "", $1)
    split($0, arr, " ")
    coverage = arr[length(arr)]
    covnum = coverage
    gsub("%", "", covnum)

    if (covnum >= 95.0) emoji = "🟢"
    else if (covnum >= 80.0) emoji = "🟡"
    else emoji = "🟠"

    printf "  %s  %-45s %s\n", emoji, substr($1":"arr[2], 1, 45), coverage
  }'

echo ""
echo "┌─────────────────────────────────────────────────────────────────┐"
echo "│  ⚠️  Áreas que Necesitan Más Tests                              │"
echo "└─────────────────────────────────────────────────────────────────┘"
echo ""

LOW_COV=$(go test ./... -cover 2>&1 | \
  grep "coverage:" | \
  grep -v "0.0%" | \
  grep -E '[0-6][0-9]\.[0-9]%' | \
  grep -v "cmd/" | \
  grep -v "gen/" | \
  wc -l | tr -d ' ')

if [ "$LOW_COV" -eq 0 ]; then
  echo "  ✅ ¡Excelente! Todos los paquetes tienen buena cobertura (>60%)"
else
  go test ./... -cover 2>&1 | \
    grep "coverage:" | \
    grep -v "0.0%" | \
    grep -E '[0-6][0-9]\.[0-9]%' | \
    grep -v "cmd/" | \
    grep -v "gen/" | \
    awk '{
      pkg = $2
      gsub("github.com/cmelgarejo/go-modulith-template/", "", pkg)
      cov = $(NF-2)
      printf "  🔴  %-50s %s\n", pkg, cov
    }'
fi

echo ""
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║  💡 Comandos Útiles                                            ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""
echo "  📊 Ver reporte HTML:     just test-coverage"
echo "  🧪 Ejecutar tests:       just test"
echo "  📈 Este reporte:         just coverage-report"
echo "  🌐 Abrir en navegador:   just coverage-html"
echo ""

