# Secrets Management Guide for Melodee

## Overview
This file documents the secrets management approach for Melodee in production environments.

## Secrets Storage Options

### 1. Kubernetes Secrets (Recommended for K8s deployments)
- Store sensitive data as Kubernetes secrets
- Mount secrets as environment variables or files
- Example:
  ```yaml
  apiVersion: v1
  kind: Secret
  metadata:
    name: melodee-secrets
  type: Opaque
  data:
    db-password: <base64-encoded-password>
    jwt-secret: <base64-encoded-jwt-secret>
  ```

### 2. HashiCorp Vault (Recommended for high-security environments)
- Use vault-agent for automatic secret injection
- Implement short-lived tokens with renewal
- Configure secret rotation policies

### 3. AWS Secrets Manager / Azure Key Vault / GCP Secret Manager
- Cloud-native secret storage and rotation
- IAM-based access control
- Audit logging built-in

## Required Secrets

### Database
- `MELODEE_DB_PASSWORD`: Database user password
- `POSTGRES_PASSWORD`: PostgreSQL superuser password (if applicable)

### Authentication & Security
- `MELODEE_JWT_SECRET`: Secret key for JWT signing (min 32 chars)
- `MELODEE_CRYPTO_KEY`: Symmetric encryption key for sensitive data

### External Services
- `LASTFM_API_KEY`: LastFM API key for metadata scraping
- `MUSICBRAINZ_API_KEY`: MusicBrainz API key
- `SPOTIFY_CLIENT_ID`: Spotify client ID
- `SPOTIFY_CLIENT_SECRET`: Spotify client secret
- `GRAFANA_ADMIN_PASSWORD`: Grafana admin password

## Security Best Practices
- Never commit secrets to version control
- Use different secrets for each environment (dev, staging, prod)
- Rotate secrets regularly (especially JWT secret)
- Use least-privilege principle for database users
- Monitor secret access and rotation
- Enforce strong password policies for all secrets

## Docker Compose Development
For local development, use .env file with strong passwords (not committed to git)