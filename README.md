# Typewriterpost

Typewriterpost is a modern, cloud-native email platform. This repository currently contains the **initial JMAP API scaffold** plus documentation and deployment helpers that align with the long-term vision in `docs/architecture/vision.md`.

## Current Features (Scaffold)
- **JMAP API service (Go)** with graceful shutdown handling.
- **Health endpoint**: `GET /healthz` returns `ok`.
- **JMAP discovery endpoint**: `GET /.well-known/jmap` returns a minimal discovery document pointing clients to `/jmap`.
- **Admin portal**: `GET /admin` provides a lightweight HTML portal (localhost-only).
- **Admin status**: `GET /admin/status` returns a JSON snapshot for quick validation.
- **Global admin center APIs**:
  - `GET /admin/api/tenants` for tenant listings (stub data today).
  - `GET /admin/api/tenants/{tenantId}/users` for tenant user listings (stub data today).
- **Tenant management portals**:
  - `GET /admin/tenants/{tenantId}` for global admin tenant management.
  - `GET /tenant/{tenantId}` for tenant-scoped administration (same settings per tenant).
  - `GET /tenant/{tenantId}/login` for tenant admin sign-in.
- **Environment-based configuration**:
  - `JMAP_API_ADDRESS` (default `:8080`)
  - `JMAP_LOG_LEVEL` (default `info`)
- **Systemd-friendly deployment** for Ubuntu via `scripts/deploy-ubuntu.sh`.
- **Baseline mail transport** installed by the script (Postfix + TLS) for SMTP ingress/egress. Mailbox storage is intended to be JMAP-native.
- **Optional webmail**: Roundcube can be installed via `--with-roundcube` for future IMAP gateway compatibility.

> Note: This is the first building block. JMAP Mail methods, auth, persistence, and push are intentionally not implemented yet. See the roadmap in `docs/architecture/vision.md`.

## Requirements
- Go 1.22+ (for local builds)
- Ubuntu 22.04+ (for the provided deployment script)

## Local Development
```bash
go build ./cmd/jmap-api
./jmap-api
```

Verify:
```bash
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/.well-known/jmap
```

## Ubuntu Deployment (single node)
A scripted installer is available for quickly standing up the JMAP API on Ubuntu.

### Install (fresh or update-in-place)
```bash
sudo ./scripts/deploy-ubuntu.sh
```

### Clean reinstall (remove old install + systemd unit)
```bash
sudo ./scripts/deploy-ubuntu.sh --clean
```

### Optional Roundcube install (requires future IMAP gateway)
```bash
sudo ./scripts/deploy-ubuntu.sh --with-roundcube
```

### Verify service
```bash
curl http://127.0.0.1:8080/healthz
curl http://127.0.0.1:8080/.well-known/jmap
curl http://127.0.0.1:8080/admin
curl http://127.0.0.1:8080/admin/status
curl http://127.0.0.1:8080/admin/api/tenants
curl http://127.0.0.1:8080/admin/api/tenants/default/users
curl http://127.0.0.1:8080/admin/tenants/default
curl http://127.0.0.1:8080/tenant/default
curl http://127.0.0.1:8080/tenant/default/login
```

Admin portal routes (`/admin/...`) are available only on localhost.

## Configuration
Edit `/etc/typewriterpost/jmap-api.env` and restart the service:
```bash
sudo systemctl restart typewriterpost-jmap-api
```

Mail server defaults are written to `/etc/typewriterpost/mail.env` and used to configure Postfix + Dovecot:
- `MAIL_DOMAIN`
- `MAIL_HOSTNAME`
- `MAIL_TLS_CERT`
- `MAIL_TLS_KEY`

To change mail server values, update the file and re-run the deploy script (or adjust Postfix/Dovecot configs directly).

## Documentation
- Vision & architecture: `docs/architecture/vision.md`

## License
All rights reserved. See `LICENSE`.
