#!/usr/bin/env bash
set -euo pipefail

show_help() {
  cat <<'USAGE'
Typewriterpost Ubuntu deployment script

Usage:
  sudo ./scripts/deploy-ubuntu.sh [--clean]

Options:
  --clean   Remove existing service/unit and install directories before reinstalling.
USAGE
}

CLEAN_INSTALL=false

for arg in "$@"; do
  case "$arg" in
    --clean)
      CLEAN_INSTALL=true
      ;;
    -h|--help)
      show_help
      exit 0
      ;;
    *)
      echo "Unknown argument: $arg" >&2
      show_help
      exit 1
      ;;
  esac
done

if [[ "${EUID}" -ne 0 ]]; then
  echo "This script must be run as root (use sudo)." >&2
  exit 1
fi

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INSTALL_DIR="/opt/typewriterpost"
CONFIG_DIR="/etc/typewriterpost"
DATA_DIR="/var/lib/typewriterpost"
MAIL_CERT_DIR="/etc/ssl/typewriterpost"
SERVICE_NAME="typewriterpost-jmap-api"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
ENV_FILE="${CONFIG_DIR}/jmap-api.env"
MAIL_ENV_FILE="${CONFIG_DIR}/mail.env"

if [[ ! -f "${REPO_ROOT}/go.mod" ]]; then
  echo "Run this script from the Typewriterpost repo root." >&2
  exit 1
fi

if [[ "${CLEAN_INSTALL}" == "true" ]]; then
  echo "Performing clean install..."
  systemctl stop "${SERVICE_NAME}" >/dev/null 2>&1 || true
  systemctl disable "${SERVICE_NAME}" >/dev/null 2>&1 || true
  rm -f "${SERVICE_FILE}"
  rm -rf "${INSTALL_DIR}"
  rm -rf "${CONFIG_DIR}"
  rm -rf "${DATA_DIR}"
  rm -rf "${MAIL_CERT_DIR}"
  systemctl daemon-reload
fi

apt-get update -y
apt-get install -y --no-install-recommends \
  ca-certificates \
  curl \
  golang-go \
  openssl \
  postfix \
  dovecot-imapd

if ! id -u typewriterpost >/dev/null 2>&1; then
  useradd --system --create-home --home-dir /var/lib/typewriterpost --shell /usr/sbin/nologin typewriterpost
fi

mkdir -p "${INSTALL_DIR}" "${CONFIG_DIR}" "${DATA_DIR}" "${MAIL_CERT_DIR}"
chown -R typewriterpost:typewriterpost "${DATA_DIR}"

cd "${REPO_ROOT}"

echo "Building JMAP API..."
go build -o "${INSTALL_DIR}/jmap-api" ./cmd/jmap-api

if [[ ! -f "${ENV_FILE}" ]]; then
  cat <<'ENV' > "${ENV_FILE}"
JMAP_API_ADDRESS=:8080
JMAP_LOG_LEVEL=info
ENV
fi

if [[ ! -f "${MAIL_ENV_FILE}" ]]; then
  MAIL_DOMAIN="$(hostname -d 2>/dev/null || true)"
  if [[ -z "${MAIL_DOMAIN}" ]]; then
    MAIL_DOMAIN="example.local"
  fi
  MAIL_HOSTNAME="$(hostname -f 2>/dev/null || true)"
  if [[ -z "${MAIL_HOSTNAME}" ]]; then
    MAIL_HOSTNAME="mail.${MAIL_DOMAIN}"
  fi
  cat <<ENV > "${MAIL_ENV_FILE}"
MAIL_DOMAIN=${MAIL_DOMAIN}
MAIL_HOSTNAME=${MAIL_HOSTNAME}
MAIL_TLS_CERT=${MAIL_CERT_DIR}/mail.crt
MAIL_TLS_KEY=${MAIL_CERT_DIR}/mail.key
ENV
fi

source "${MAIL_ENV_FILE}"

if [[ ! -f "${MAIL_TLS_CERT}" || ! -f "${MAIL_TLS_KEY}" ]]; then
  openssl req -x509 -nodes -newkey rsa:4096 -days 365 \
    -subj "/CN=${MAIL_HOSTNAME}" \
    -keyout "${MAIL_TLS_KEY}" \
    -out "${MAIL_TLS_CERT}"
fi

cat <<EOF > /etc/postfix/main.cf
myhostname = ${MAIL_HOSTNAME}
mydomain = ${MAIL_DOMAIN}
myorigin = \$mydomain
inet_interfaces = all
inet_protocols = all
mydestination = \$myhostname, localhost.\$mydomain, localhost, \$mydomain
home_mailbox = Maildir/
smtpd_banner = \$myhostname ESMTP Typewriterpost
smtpd_tls_cert_file = ${MAIL_TLS_CERT}
smtpd_tls_key_file = ${MAIL_TLS_KEY}
smtpd_tls_security_level = may
smtp_tls_security_level = may
smtpd_tls_auth_only = yes
compatibility_level = 3.6
EOF

cat <<'EOF' > /etc/dovecot/conf.d/10-mail.conf
mail_location = maildir:~/Maildir
EOF

cat <<'EOF' > /etc/dovecot/conf.d/10-auth.conf
disable_plaintext_auth = yes
auth_mechanisms = plain login
!include auth-system.conf.ext
EOF

cat <<EOF > /etc/dovecot/conf.d/10-ssl.conf
ssl = required
ssl_cert = <${MAIL_TLS_CERT}
ssl_key = <${MAIL_TLS_KEY}
EOF

cat <<'UNIT' > "${SERVICE_FILE}"
[Unit]
Description=Typewriterpost JMAP API
After=network.target

[Service]
Type=simple
User=typewriterpost
Group=typewriterpost
EnvironmentFile=/etc/typewriterpost/jmap-api.env
WorkingDirectory=/opt/typewriterpost
ExecStart=/opt/typewriterpost/jmap-api
Restart=on-failure
RestartSec=5
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/typewriterpost

[Install]
WantedBy=multi-user.target
UNIT

systemctl daemon-reload
systemctl enable "${SERVICE_NAME}"
systemctl restart "${SERVICE_NAME}"

systemctl status "${SERVICE_NAME}" --no-pager
systemctl restart postfix
systemctl restart dovecot

echo "Deployment complete."
echo "Health check: curl http://127.0.0.1:8080/healthz"
echo "Admin portal: http://127.0.0.1:8080/admin"
echo "Tenants API: http://127.0.0.1:8080/admin/api/tenants"
echo "Tenant users API: http://127.0.0.1:8080/admin/api/tenants/default/users"

for url in \
  "http://127.0.0.1:8080/healthz" \
  "http://127.0.0.1:8080/.well-known/jmap" \
  "http://127.0.0.1:8080/admin" \
  "http://127.0.0.1:8080/admin/status" \
  "http://127.0.0.1:8080/admin/api/tenants" \
  "http://127.0.0.1:8080/admin/api/tenants/default/users"; do
  status="$(curl -s -o /dev/null -w "%{http_code}" "${url}" || true)"
  if [[ "${status}" == "200" ]]; then
    echo "OK ${url}"
  else
    echo "WARN ${url} returned ${status}"
  fi
done
