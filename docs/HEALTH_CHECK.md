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
- Export `melodee_health_status{dependency="db|redis"}` gauge (1=ok, 0=down).
- Units: `latency_ms` in milliseconds.
