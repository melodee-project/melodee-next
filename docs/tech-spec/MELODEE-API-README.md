# melodee-api — Combined API, Workers, Infra/Ops, and Auth Config

This README captures the proposed repo layout, build artifacts, and run conventions for the combined server repository.

## Purpose
- Host the HTTP API (Go + Gin) implementing OpenSubsonic and first-party extras
- Contain workers (scanner/tagger, transcoder, indexer)
- Keep infra (Terraform/Helm), ops (dashboards, k6), and IdP configuration co-located

## Repository structure
```
/cmd/
  api/
  worker-scanner/
  worker-transcoder/
  worker-indexer/
/internal/
/pkg/
/migrations/
/ops/
  dashboards/
  alerts/
  k6/
/infra/
  terraform/
  helm/
/auth/
/deploy/
```

## Build artifacts
- Containers: api, worker-scanner, worker-transcoder, worker-indexer
- Optional: single multi-binary image for compact deployments

## Config & environments
- Config via environment variables and/or config files (12-factor)
- Environments: dev, staging, prod; overlays in /deploy or Helm values files
- Secrets via Vault/KMS or sealed secrets; no secrets in images

## Local development
- Dependencies: Go 1.22+, Docker, Make
- Recommended: docker-compose for Postgres, Redis, MinIO, Keycloak (optional)
- Run targets (illustrative):
  - make dev-api
  - make dev-scanner
  - make dev-transcoder
  - make dev-indexer
- Seed/demo data scripts optional in /deploy or /ops

## CI/CD
- Lint, test, build binaries and images for api and workers
- Validate Terraform and Helm with dry-runs; publish images tagged by SHA and semver
- Contract tests against melodee-specs OpenAPI; publish SBOM/signatures

## Observability
- OTel exporter config via env; Prometheus metrics endpoint on all services
- Dashboards and alerts in /ops

## Security
- TLS termination at edge (Caddy/NGINX); HSTS
- RBAC and scope checks at API boundary; short-lived signed HLS URLs
- Audit trails for admin actions; DSAR-friendly exports
