# Observability Setup Guide

This guide explains how to set up and configure observability for the modulith template, including Prometheus, Grafana, and logging.

## Table of Contents

1. [Overview](#overview)
2. [Local Development Setup](#local-development-setup)
3. [Prometheus Configuration](#prometheus-configuration)
4. [Grafana Configuration](#grafana-configuration)
5. [Dashboard Setup](#dashboard-setup)
6. [Alert Configuration](#alert-configuration)
7. [Production Setup](#production-setup)
8. [Troubleshooting](#troubleshooting)

## Overview

The observability stack includes:

- **Prometheus**: Metrics collection and storage
- **Grafana**: Metrics visualization and dashboards
- **Structured Logging**: JSON-formatted logs (slog)
- **OpenTelemetry**: Distributed tracing (optional, via Jaeger)

### Metrics Exposed

The application exposes metrics at `/metrics` endpoint:

- HTTP request metrics (rate, latency, status codes)
- gRPC metrics (rate, latency, status codes)
- Database metrics (connections, queries, cache hit ratio)
- Event metrics (publish rate, handler errors)
- System metrics (CPU, memory, goroutines)

## Local Development Setup

### Docker Compose

The observability stack is included in `docker-compose.yaml`:

```bash
# Start all services including observability
docker-compose up -d

# Or start only observability services
docker-compose up -d prometheus grafana
```

Services are available at:

- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)
- **Jaeger**: http://localhost:16686 (if enabled)

### Verify Setup

```bash
# Check Prometheus is scraping metrics
curl http://localhost:9090/api/v1/targets

# Check application metrics endpoint
curl http://localhost:8000/metrics

# Check Grafana is running
curl http://localhost:3000/api/health
```

## Prometheus Configuration

### Configuration File

Prometheus configuration is in `deployment/prometheus.yaml`:

```yaml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'modulith-server'
    static_configs:
      - targets: ['server:8000']
    metrics_path: '/metrics'
```

### Adding Alert Rules

Alert rules are in `deployment/prometheus/alerts/rules.yaml`. To use them, update `prometheus.yaml`:

```yaml
rule_files:
  - 'alerts/rules.yaml'  # Relative to prometheus config file
```

### Custom Metrics

To add custom metrics, use the telemetry package:

```go
import "github.com/LoopContext/go-modulith-template/internal/telemetry"

// Register custom counter
telemetry.Counter("custom_operations_total", "Description")

// Register custom histogram
telemetry.Histogram("custom_duration_seconds", "Description")
```

## Grafana Configuration

### Default Credentials

- **Username**: `admin`
- **Password**: `admin` (change on first login)

### Data Source Setup

1. Go to Configuration → Data Sources
2. Add Prometheus data source
3. URL: `http://prometheus:9090` (for Docker) or `http://localhost:9090` (for local)
4. Click "Save & Test"

### Dashboard Import

Dashboards are provided in `deployment/grafana/dashboards/`:

1. Go to Dashboards → Import
2. Upload JSON file or paste JSON content
3. Select Prometheus data source
4. Click "Import"

Available dashboards:

- **Application Overview** (`application-overview.json`)
  - HTTP request metrics
  - Request rate, latency, error rate
  - Status code distribution

- **gRPC Overview** (`grpc-overview.json`)
  - gRPC request metrics
  - Request rate, latency, error rate
  - Status code distribution

- **Database Overview** (`database-overview.json`)
  - Database connections
  - Query rate and latency
  - Cache hit ratio
  - Database size

- **Events Overview** (`events-overview.json`)
  - Event publish rate
  - Event handler duration and errors
  - Event queue size

- **Modules Overview** (`modules-overview.json`)
  - Per-module metrics
  - Module request rate and errors
  - Module events

### Dashboard Customization

Dashboards can be customized in Grafana:

1. Open dashboard
2. Click "Edit" (gear icon)
3. Modify panels, queries, or variables
4. Click "Save"

## Alert Configuration

### Prometheus Alerts

Alert rules are defined in `deployment/prometheus/alerts/rules.yaml`:

- **Application Alerts**: High error rate, latency, low success rate
- **gRPC Alerts**: High error rate, latency, low success rate
- **Database Alerts**: High connections, low cache hit ratio, slow queries
- **Event Alerts**: High handler error rate, large queue
- **System Alerts**: Service down, high memory/CPU usage

### Alerting Channels

Configure alerting channels in Grafana:

1. Go to Alerting → Notification channels
2. Add channel (Email, Slack, PagerDuty, etc.)
3. Configure channel settings
4. Test notification

### Alert Rules

To add custom alert rules, edit `deployment/prometheus/alerts/rules.yaml`:

```yaml
groups:
  - name: custom_alerts
    interval: 30s
    rules:
      - alert: CustomAlert
        expr: rate(custom_metric[5m]) > 10
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Custom alert triggered"
          description: "Metric value is {{ $value }}"
```

Reload Prometheus configuration:

```bash
# Send SIGHUP to reload
kill -HUP $(pidof prometheus)

# Or restart container
docker-compose restart prometheus
```

## Production Setup

### Kubernetes Deployment

For Kubernetes deployment, use Helm charts in `deployment/helm/`.

#### Prometheus

1. Install Prometheus operator (if not installed):
   ```bash
   helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
   helm install prometheus prometheus-community/kube-prometheus-stack
   ```

2. Create ServiceMonitor for application:
   ```yaml
   apiVersion: monitoring.coreos.com/v1
   kind: ServiceMonitor
   metadata:
     name: modulith-server
   spec:
     selector:
       matchLabels:
         app: modulith-server
     endpoints:
       - port: http
         path: /metrics
   ```

#### Grafana

1. Import dashboards via ConfigMap:
   ```bash
   kubectl create configmap grafana-dashboards \
     --from-file=deployment/grafana/dashboards/
   ```

2. Mount in Grafana deployment:
   ```yaml
   volumeMounts:
     - name: dashboards
       mountPath: /etc/grafana/provisioning/dashboards
   volumes:
     - name: dashboards
       configMap:
         name: grafana-dashboards
   ```

### Environment Variables

Configure observability via environment variables:

```bash
# Metrics endpoint
METRICS_ENABLED=true
METRICS_PATH=/metrics
METRICS_PORT=8000

# Log level
LOG_LEVEL=info  # debug, info, warn, error

# Log format
LOG_FORMAT=json  # json, text
```

### Log Aggregation

For production, use a log aggregation system:

- **ELK Stack**: Elasticsearch, Logstash, Kibana
- **Loki**: Grafana Loki (lightweight, integrates with Grafana)
- **CloudWatch**: AWS CloudWatch Logs
- **Datadog**: Datadog Logs

#### Loki Setup (Example)

```yaml
# docker-compose.yaml
services:
  loki:
    image: grafana/loki:latest
    ports:
      - "3100:3100"

  promtail:
    image: grafana/promtail:latest
    volumes:
      - ./logs:/var/log/app
      - ./promtail-config.yaml:/etc/promtail/config.yaml
```

## Troubleshooting

### Metrics Not Appearing

1. **Check metrics endpoint**:
   ```bash
   curl http://localhost:8000/metrics
   ```

2. **Check Prometheus targets**:
   - Go to http://localhost:9090/targets
   - Verify target is UP

3. **Check scrape configuration**:
   - Verify `prometheus.yaml` has correct target
   - Check network connectivity

### Dashboards Not Loading

1. **Check data source**:
   - Verify Prometheus data source is configured
   - Test data source connection

2. **Check queries**:
   - Verify metric names match actual metrics
   - Check Prometheus for available metrics

3. **Check dashboard JSON**:
   - Validate JSON syntax
   - Verify schema version compatibility

### Alerts Not Firing

1. **Check alert rules**:
   ```bash
   curl http://localhost:9090/api/v1/rules
   ```

2. **Check alert evaluation**:
   - Go to http://localhost:9090/alerts
   - Verify alerts are evaluated

3. **Check alert expression**:
   - Test expression in Prometheus query UI
   - Verify threshold values

### High Memory Usage

1. **Reduce scrape interval** (if needed):
   ```yaml
   global:
     scrape_interval: 30s  # Increase from 15s
   ```

2. **Reduce retention**:
   ```yaml
   storage:
     retention: 7d  # Reduce from default
   ```

3. **Filter metrics**:
   - Use metric relabeling to drop unused metrics
   - Use recording rules to pre-aggregate metrics

## Best Practices

1. **Metric Naming**:
   - Use consistent naming convention
   - Include units in metric names (seconds, bytes, etc.)
   - Use labels for dimensions

2. **Dashboard Design**:
   - Start with high-level overview
   - Drill down into details
   - Use consistent colors and units

3. **Alert Design**:
   - Set appropriate thresholds
   - Use multiple severity levels
   - Include runbooks in annotations

4. **Logging**:
   - Use structured logging (JSON in production)
   - Include request IDs for correlation
   - Don't log sensitive information

5. **Performance**:
   - Monitor metric cardinality
   - Use recording rules for expensive queries
   - Set appropriate scrape intervals

## Summary

- Use Docker Compose for local development
- Configure Prometheus to scrape application metrics
- Import Grafana dashboards for visualization
- Set up alert rules for critical conditions
- Use structured logging with request IDs
- Monitor metrics cardinality and performance

For more information, see:
- [Logging Standards](LOGGING_STANDARDS.md)
- [Prometheus Documentation](https://prometheus.io/docs/)
- [Grafana Documentation](https://grafana.com/docs/)
- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)

