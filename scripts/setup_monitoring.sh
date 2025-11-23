#!/bin/bash

# Monitoring Configuration Setup Script
# Sets up Prometheus and Grafana configurations for SLO-based monitoring

echo "Setting up monitoring dashboards and configurations..."

# Create prometheus config with appropriate metrics collection
cat > /etc/prometheus/melodee.yml << 'EOF'
# Prometheus configuration for Melodee application
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "melodee_rules.yml"

scrape_configs:
  - job_name: 'melodee-api'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: /metrics
    scrape_interval: 10s
    scrape_timeout: 5s

  - job_name: 'melodee-processing'
    static_configs:
      - targets: ['localhost:8081']
    metrics_path: /metrics
    scrape_interval: 15s
    scrape_timeout: 5s
EOF

# Create alerting rules for SLO monitoring
cat > /etc/prometheus/melodee_rules.yml << 'EOF'
groups:
  - name: melodee.rules
    rules:
      # Availability SLO: Ensure service is accessible
      - alert: ServiceDown
        expr: up{job="melodee-api"} == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "Melodee API is down"
          description: "Melodee API has been down for more than 2 minutes"

      # Error Budget Burn Rate: Track error rate SLO violations
      - alert: HighErrorRate
        expr: (sum(rate(http_requests_total{status=~"5.."}[5m])) / sum(rate(http_requests_total[5m]))) > 0.005
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
          description: "Error rate is above 0.5% for more than 5 minutes: {{ $value | printf \"%.2f\" }}%"

      # Latency SLO: Track request latency
      - alert: HighLatency
        expr: histogram_quantile(0.95, sum by(le) (rate(http_request_duration_seconds_bucket[5m]))) > 1.0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High request latency detected"
          description: "95th percentile latency is above 1 second: {{ $value }}s"

      # Capacity SLO: Monitor storage capacity
      - alert: HighCapacityUsage
        expr: melodee_capacity_percent > 90
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High capacity usage detected"
          description: "Capacity usage is above 90%: {{ $value }}% for path {{ $labels.path }}"

      # Queue Depth SLO: Monitor processing queue sizes
      - alert: HighQueueDepth
        expr: melodee_processing_queue_length > 100
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High queue depth detected"
          description: "Processing queue depth is above 100: {{ $value }} for queue {{ $labels.type }}"
EOF

# Create dashboards directory structure
mkdir -p /var/lib/grafana/dashboards/melodee/

# Copy the enhanced dashboard
cp ./monitoring/dashboards/melodee.json /var/lib/grafana/dashboards/melodee/

# Create Grafana provisioning configuration
mkdir -p /etc/grafana/provisioning/dashboards
mkdir -p /etc/grafana/provisioning/datasources

cat > /etc/grafana/provisioning/datasources/melodee.yml << 'EOF'
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    orgId: 1
    url: http://localhost:9090
    basicAuth: false
    isDefault: true
    editable: true
EOF

cat > /etc/grafana/provisioning/dashboards/melodee.yml << 'EOF'
apiVersion: 1

providers:
  - name: 'Melodee Dashboards'
    orgId: 1
    folder: 'Melodee'
    type: file
    disableDeletion: false
    editable: true
    options:
      path: /var/lib/grafana/dashboards/melodee
EOF

echo "Monitoring configuration completed."
echo "Dashboard panels now include:"
echo "  - System Availability (SLO: >99.9%)"
echo "  - API Error Rate (SLO: <0.1%)"
echo "  - API Latency p95 (SLO: <500ms)"
echo "  - Queue Depths and Processing Metrics"
echo "  - Database Connection Monitoring"
echo "  - API Request Rate Tracking"
echo "  - Capacity Monitoring by Library"
echo "  - Processing Queue Lengths"
echo "  - System Resource Utilization"
echo "  - Library Processing Pipeline Status"