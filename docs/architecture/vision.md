# Typewriterpost Vision & Architecture Overview

## Table of Contents
1. [Mission & Product Positioning](#mission--product-positioning)
2. [Protocols & Compatibility Strategy](#protocols--compatibility-strategy)
3. [Service Architecture](#service-architecture)
   1. [Core Services](#core-services)
   2. [Supporting Platforms](#supporting-platforms)
   3. [Cross-Cutting Concerns](#cross-cutting-concerns)
4. [Data & Storage Design](#data--storage-design)
5. [Security Baseline](#security-baseline)
6. [Web Applications & Experience](#web-applications--experience)
7. [API Surface](#api-surface)
8. [Mail Deliverability & Filtering](#mail-deliverability--filtering)
9. [Deployment & Operations](#deployment--operations)
10. [Testing & Quality Strategy](#testing--quality-strategy)
11. [Roadmap](#roadmap)
12. [Repository Layout](#repository-layout)
13. [UI Styleguide Snapshot](#ui-styleguide-snapshot)
14. [Licensing & Compliance](#licensing--compliance)
15. [CI/CD Reference Pipeline](#cicd-reference-pipeline)
16. [Acceptance Criteria](#acceptance-criteria)
17. [Implementation Jumpstart](#implementation-jumpstart)

---

## Mission & Product Positioning
- **Mission:** Deliver a modern, cloud-native e-mail platform with real-time synchronisation, Exchange-grade reliability, and a luxurious 1820s/“Peaky Blinders” aesthetic wrapped in an accessible, contemporary UX.
- **User Promise:** "Exchange-like stability + Apple-like fluidity, yet simpler, faster, and safer than classic IMAP/POP/SMTP-only stacks."
- **Version 1 Scope:**
  - Full-featured mail host including web client, accounts, aliases, and domain management.
  - Real-time push synchronisation with multi-device conflict resolution.
  - Calendar and contacts with synchronisation.
  - Compatibility layers for legacy clients plus a modern API surface for custom clients.
  - Multi-tenant deployment that can scale from single-node to clustered installations behind a load balancer.

## Protocols & Compatibility Strategy
- **Primary Protocol:** JMAP (mail, calendar, contacts) for efficient batch operations, push support, and cloud-first semantics.
- **Mandatory Delivery:** SMTP remains the internet-standard transport. LMTP/SMTP is used internally for store-and-forward toward mailbox storage.
- **Compatibility Bridges:**
  - IMAP/POP gateway for legacy clients (read-only in v1).
  - Optional ActiveSync proxy (e.g., Z-Push) with attention to AGPL licensing, or a bespoke minimal subset proxy.
  - Optional CalDAV/CardDAV gateway when JMAP-aware calendar/contacts clients are missing.
  - Server-side filtering using Sieve (ManageSieve) with a user-friendly editor.
- **Design Directive:** JMAP-first. Gateways are feature-flagged extensions to avoid dragging legacy complexity into the core services.

## Service Architecture

### Core Services
| Service | Responsibilities | Technology | Notes |
| --- | --- | --- | --- |
| Auth & Identity | OIDC/OAuth2 flows, WebAuthn/FIDO2, TOTP, device registration, RBAC enforcement | Go + PostgreSQL | Multi-tenant policies, per-tenant branding, audit trails |
| Account & Tenant Management | Tenant lifecycle, domain onboarding, quotas, alias management, DNS helpers | Go + PostgreSQL | Integrates with DNS APIs, issues DKIM keys |
| Mail Ingress/Egress (MTA) | SMTP/LMTP endpoints, queueing, TLS, MTA-STS/DANE, DKIM signing, bounce handling | Go (mox-based stack) | Hooks into spam/AV pipeline and delivery workers |
| JMAP API | Mailbox state, change tracking, blob upload/download, JMAP push | Go | Stateless API, leverages event bus for state changes |
| Calendar & Contacts | JMAP calendars, invitations, availability, shared resources | Go | Targets v1.2 milestone |
| Webmail | Next.js/React client, offline-first caching, keyboard shortcuts, accessibility | TypeScript | Shares UI library with admin console |

### Supporting Platforms
- **Storage:**
  - Metadata in PostgreSQL (HA via Patroni or Aurora).
  - Message blobs and attachments in an S3-compatible object store (MinIO for dev, S3 in prod) with optional NVMe cache.
  - Full-text search indexes powered by OpenSearch/Elasticsearch or Meilisearch (pluggable).
- **Messaging & Queues:** NATS (preferred) or Kafka for fan-out of indexing, push notifications, and background jobs.
- **Caching:** Redis for sessions, rate limits, idempotency keys, and JMAP state caching.
- **Observability Stack:** OpenTelemetry traces, Prometheus metrics, Grafana dashboards, Loki logs, backed by structured audit logging.

### Cross-Cutting Concerns
- **Notifications:** WebSocket hub and Web Push dispatcher for JMAP state updates and mobile push webhooks.
- **Anti-spam/AV:** rspamd with Bayes/RBL/fuzzy pipelines, ClamAV scanning, SPF/DKIM/DMARC/ARC processing, optional BIMI.
- **Gateway/Edge:** API gateway with JWT validation, WAF, rate limiting, per-tenant mTLS for intra-service traffic.

## Data & Storage Design
Core entities and relationships:
- `User(id, tenant_id, email, password_hash/null, webauthn_public_keys[], roles[], settings_json)`
- `Mailbox(id, user_id, name, role, counters, retention_policy)`
- `MessageMeta(id, mailbox_id, thread_id, from, to[], cc[], bcc[], subject, date, flags, labels[], size, blob_id, dkim_status, spam_score, encryption_state)`
- `Blob(id, location, checksum, size, encryption_key_ref)`
- `Thread(id, last_message_date, participants[], subject_hint)`
- `Rule(id, user_id, sieve_source, enabled, order)`
- `Domain(id, tenant_id, name, dkim_key_ref, dmarc_policy, verified, spf_mode)`
- `Device(id, user_id, device_fingerprint, last_seen, push_token, client_capabilities)`
- `AuditLog(id, actor, action, target, ts, ip, metadata)`

Design considerations:
- Separate state/change tables per JMAP specification to enable efficient delta streaming and optimistic concurrency.
- Store encryption metadata alongside blobs to support mailbox-level keys and potential future E2EE.
- Maintain event streams (append-only) for auditability and to drive downstream indexing/search pipelines.

## Security Baseline
- **Transport:** Enforce TLS 1.2+ across all protocols, with HSTS, MTA-STS, TLSRPT, and DANE where feasible.
- **Content Protection:** DKIM signing, DMARC enforcement, ARC for forwarders, optional BIMI.
- **Authentication:** WebAuthn, TOTP, OAuth2 (PKCE), per-app tokens, device binding.
- **Encryption:** At-rest encryption through KMS-managed keys, mailbox-level key scopes, server-side re-encryption during rotations.
- **Abuse & Rate-limiting:** IP/org-based throttling, tarpitting, greylisting controls, fail2ban integration.
- **Isolation & Governance:** Per-tenant key scopes, RBAC (user/admin/auditor), zero-trust mTLS for service-to-service traffic.
- **Privacy & Compliance:** GDPR-aligned data minimisation, export/delete tooling, retention policies, full audit trail.
- **Supply Chain:** SBOM generation (Syft), image signing (Cosign), dependency scanning (Dependabot/Renovate), secret scanning.

## Web Applications & Experience
- **Design Language:** Modern layout infused with 1820s luxury aesthetics—Didot-inspired typography, copper/ebony accents, subtle paper textures—while preserving WCAG 2.1 AA contrast ratios.
- **Core Components:**
  - Mailbox: Three-pane layout (folders, thread list, conversation view) with optional Vim-like shortcuts.
  - Composer: Autosave drafts, offline-first behaviour, drag-and-drop/clipboard attachments, templating support.
  - Search: Low-latency full-text queries with filters (from:, to:, has:attachment, date ranges).
  - Rules Builder: Visual block editor plus advanced Sieve code tab.
  - Admin Console: Tenant/domain wizard, DNS checks, DKIM key management, deliverability diagnostics.
  - Calendar & Contacts: JMAP-driven scheduling, invitations, resource booking.
- **Accessibility:** RTL layouts, comprehensive hotkeys, "reduce motion" toggle, screen-reader-tested navigation.

## API Surface
- **`/jmap`:** Batched JMAP Mail endpoints with push state tokens; later phases add calendar and contacts.
- **`/auth`:** OAuth2/OIDC flows, device code, token introspection, WebAuthn ceremonies.
- **`/admin`:** RBAC-protected tenant/domain management, deliverability insights, metrics snapshots.
- **Webhooks:** Delivery status, spam verdict updates, out-of-office status changes.
- **Push:** WebSocket endpoint `/push` for low-latency JMAP state updates plus Web Push integration for PWAs.

## Mail Deliverability & Filtering
- **Inbound Pipeline:** rspamd (Bayes, RBLs, fuzzy hashing), SPF/DKIM/DMARC validation, ClamAV virus scanning, quarantine handling, per-tenant tuning knobs.
- **Outbound Strategy:** IP/domain warm-up guides, per-tenant reputation scoring, rate control, priority queues for transactional mail.
- **Tooling:** Seed testing harness, postmaster dashboard, feedback loop parsers for major providers.

## Deployment & Operations
- **Infrastructure as Code:** Terraform modules for VPC, load balancers, PostgreSQL, Redis, MinIO/S3, Kubernetes clusters, ingress, and certificates.
- **Packaging:** Container images per service with versioned Helm charts for Kubernetes deployments.
- **Environments:**
  - Dev: kind/minikube for local experimentation.
  - Staging: Production-like environment for pre-release validation.
  - Production: Multi-AZ, auto-scaling, managed PostgreSQL and object storage.
- **Secrets Management:** External Secrets operator backed by KMS; scheduled DKIM key rotation.
- **Backups & DR:** Point-in-time recovery for PostgreSQL, object-store versioning, tested restore runbooks, chaos drills.
- **Monitoring & Alerting:** SLO dashboards (availability, API latency percentiles, delivery success), alert routing via PagerDuty or Opsgenie.

## Testing & Quality Strategy
- **Unit & Integration Tests:** Protocol handling (JMAP, SMTP, DKIM), storage adapters, search indexing.
- **End-to-End Tests:** Playwright/Cypress suites for webmail and admin consoles, synthetic mail flow validation.
- **Interoperability:** Regression matrix covering Apple Mail, Outlook, Thunderbird, mobile clients via gateways.
- **Security Testing:** SAST, DAST, dependency scanning, secret scanning, fuzzing of SMTP/JMAP parsers, pentest hooks.
- **Performance Testing:** k6 load profiles for mail ingestion, large mailbox searches (>1M messages), push latency budgets.

## Roadmap
- **MVP (90–120 days):** Auth (OIDC/TOTP/WebAuthn), tenant/domain onboarding, SMTP/LMTP ingest, JMAP Mail (read/send), DKIM/SPF/DMARC, rspamd/ClamAV, Webmail v1 (list/read/compose/search), basic Sieve UI, observability foundations, Helm/Terraform deployment.
- **v1.1:** JMAP push + offline PWA, advanced rules builder, IMAP/JMAP importers.
- **v1.2:** JMAP calendars & contacts, meeting invites, availability, resource booking.
- **v1.3:** Legacy gateways (IMAP/POP), optional ActiveSync proxy, CalDAV/CardDAV bridge.
- **v1.4:** Mobile apps (wrapper/native JMAP clients), organisational features (shared mailboxes, delegated access).

## Repository Layout
```
typewriterpost/
  apps/
    api-gateway/
    jmap-api/
    smtp-mta/
    sieve-service/
    webmail/
    admin-console/
    notif-push/
  packages/
    proto/
    ui/
    core-lib/
  infra/
    helm/
    terraform/
    k6-loadtests/
  ops/
    runbooks/
    sre/
  security/
    threat-models/
    policies/
  docs/
    architecture/
    api/
    styleguide/
```

## UI Styleguide Snapshot
- **Typography:** Display—Didot-inspired; body—system serif fallback; numbers—tabular; code—JetBrains Mono.
- **Colour Palette:** Ivory (#F8F5EF), Ink (#0D0D0D), Copper (#B87333), Velvet Green (#0F2A1F), gold accents.
- **Components:** Soft shadowed cards (xl), 2xl rounded corners, subtle paper textures, bold focus states, <150ms hover animations, respect prefers-reduced-motion.
- **Iconography:** Line icons with gentle ink-bleed SVG filters.
- **Empty States:** "Telegram card" illustrations with concise, period-inspired copy that remains clear and modern.

## Licensing & Compliance
- **Licensing Strategy:** Private by default (“All rights reserved”). Open-source candidates under Apache-2.0; AGPLv3 for gateway code derived from AGPL projects.
- **Third-party Compliance:** Maintain `LICENSES/NOTICE`, automate SBOM creation in CI.

## CI/CD Reference Pipeline
- GitHub Actions pipeline that builds/tests services, generates SBOMs, builds and signs OCI images, runs security scans, publishes Helm charts, executes e2e smoke tests, and supports automated rollbacks.

## Acceptance Criteria
- Domain onboarding in <5 minutes with SPF/DKIM/DMARC checks passing.
- Webmail server render time <200ms and search p95 <800ms on 100k-message mailboxes.
- Real-time updates propagate across devices in ≤1s via JMAP push.
- Outbound mail passes SPF/DKIM/DMARC checks with major providers.
- Security posture includes working WebAuthn, TLS A+ report card, and zero critical SAST/DAST findings.

## Implementation Jumpstart
1. **Scaffold infrastructure & services:** Helm chart skeletons, Terraform modules (Postgres, Redis, MinIO/S3, ingress, certificates).
2. **Implement API gateway:** Authentication/authorisation middleware, rate limiting, telemetry hooks.
3. **Build JMAP mail service:** Core endpoints, state/change tracking, blob storage integration, search indexing worker.
4. **Deliver SMTP/LMTP service:** TLS, AUTH, DKIM signing, queue/retry logic, bounce processing, delivery pipeline integration.
5. **Ship webmail application:** Three-pane layout, conversation view, composer with autosave/offline-first features.
6. **Create JMAP client SDK:** Shared package (`packages/proto`) for types, validators, and request helpers.
7. **Develop Sieve service & UI:** ManageSieve server, visual rule builder, test harness.
8. **Launch admin console:** Tenant/domain wizard, DNS diagnostics, DKIM key management, deliverability dashboard.
9. **Establish observability:** OpenTelemetry exporters, Prometheus metrics, Grafana dashboards, structured audit logging.
10. **Harden security baseline:** WebAuthn flows, TOTP fallback, RBAC policies, CSP/HSTS/CSRF, secret management integration, dependency scanning.

---

**Preferred Implementation Notes:** Use Go as the primary server language (Rust optional for high-performance components). Target ≥90% unit coverage for protocol-critical modules, add contract tests for JMAP behaviour, and property-based tests for parsers. Optimise for low-latency JSON handling (e.g., gojay/sonic), robust connection pooling, and back-pressure aware processing. Provide service-level READMEs, ADRs for key decisions, and feature flags for gateways, end-to-end encryption, BIMI, and experimental search engines.
