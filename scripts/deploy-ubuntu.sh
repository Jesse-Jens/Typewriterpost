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
SERVICE_NAME="typewriterpost-jmap-api"
SERVICE_FILE="/etc/systemd/system/${SERVICE_NAME}.service"
ENV_FILE="${CONFIG_DIR}/jmap-api.env"

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
  systemctl daemon-reload
fi

apt-get update -y
apt-get install -y --no-install-recommends golang-go ca-certificates curl

if ! id -u typewriterpost >/dev/null 2>&1; then
  useradd --system --create-home --home-dir /var/lib/typewriterpost --shell /usr/sbin/nologin typewriterpost
fi

mkdir -p "${INSTALL_DIR}" "${CONFIG_DIR}" "${DATA_DIR}"
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

echo "Deployment complete."
echo "Health check: curl http://127.0.0.1:8080/healthz"
