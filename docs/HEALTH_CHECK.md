# Health Check Contract

Endpoint: `GET /healthz`

Response:
```json
{
  "status": "ok",
  "db": { "status": "ok", "latency_ms": 23 },
  "redis": { "status": "ok", "latency_ms": 10 }
}
```

Rules:
- HTTP 200 when all dependencies are `ok`; HTTP 503 when any dependency is degraded/unavailable.
- Degraded thresholds: DB ping > 200ms or Redis ping > 100ms marks dependency `degraded`; include `"status": "degraded"` and return 503.
- Headers: `Cache-Control: no-store`, `Content-Type: application/json`.

Metrics Labels:
- Export `melodee_health_status{dependency="db|redis"}` gauge (1=ok, 0=down, 2=degraded).
- Units: `latency_ms` in milliseconds.

## Prometheus Metrics and Alerts
- Metrics exported via `/metrics` endpoint on both Melodee API and OpenSubsonic servers
- Used for alerting in `monitoring/prometheus/alerts.yml`:
  - `DatabaseDown`: Triggered when `melodee_health_status{dependency="db"} == 0` for 1m
  - `DatabaseDegraded`: Triggered when `melodee_health_status{dependency="db"} == 2` for 5m
  - `RedisDown`: Triggered when `melodee_health_status{dependency="redis"} == 0` for 1m

## API Performance Metrics
- Request rate: `http_requests_total{method, handler, status}` counter
- Request duration: `http_request_duration_seconds_bucket{handler, le}` histogram
- Error rate: Percentage of requests with 5xx status codes
- Available for both Melodee API (`/api/.*` handlers) and OpenSubsonic API (`/rest/.*` handlers)
