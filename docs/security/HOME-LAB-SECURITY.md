# Home-lab Security Checklist

Keep it simple and solid. Apply these and you’ll have strong security without excess complexity.

## Accounts & auth
- Use strong admin password; consider enabling TOTP 2FA for admin.
- Create separate users for family/friends; least privilege (user/librarian vs admin).
- Prefer PATs for legacy clients; revoke unused PATs periodically.

## Network exposure
- Keep the service on your LAN by default. If you expose it to the internet:
  - Put a reverse proxy (Caddy/NGINX) with TLS in front.
  - Enable HSTS and redirect HTTP → HTTPS.
  - Forward only necessary ports.

## System hardening
- Run container as non-root; read-only filesystem where possible; minimal capabilities.
- Keep images up to date; apply updates regularly.
- Back up `/data` (DB and derived assets) on a schedule; test restore occasionally.

## API & app settings
- Enable short-lived signed HLS URLs (default).
- Scope PATs; set expirations when issuing.
- Rate limit login endpoints; lockout/backoff after repeated failures.
- Apply modest rate limits to engagement mutations (/me/*) to avoid abuse (e.g., 5–10 writes/sec per user).
- Log sign-ins and sensitive actions; keep logs local (no PII beyond necessity).

## Optional extras
- Prometheus metrics and alerts if you already run them.
- Fail2ban or reverse proxy rate limiting.
- If using OIDC: restrict redirect URIs; rotate client secrets.
