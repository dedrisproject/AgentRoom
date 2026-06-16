#!/bin/sh
# AgentRoom installer
# Usage: curl -fsSL https://raw.githubusercontent.com/dedrisproject/agentroom/main/install.sh | bash
# Pinned version: ... | bash -s -- --version v1.0.0
# Uninstall: agentroom-uninstall [--purge]
set -e

REPO="dedrisproject/agentroom"
BINARY="agentroom"
DEFAULT_PORT="8080"

# ---- Parse args ----
VERSION=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --version) VERSION="$2"; shift 2 ;;
    *) echo "Unknown option: $1" >&2; exit 1 ;;
  esac
done

# ---- Detect OS and arch ----
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$OS" in
  linux|darwin) ;;
  *) echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

case "$ARCH" in
  x86_64)       ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported arch: $ARCH" >&2; exit 1 ;;
esac

# ---- Resolve version ----
if [ -z "$VERSION" ]; then
  echo "Fetching latest release..."
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')"
  if [ -z "$VERSION" ]; then
    echo "Failed to detect latest version. Set --version explicitly." >&2
    exit 1
  fi
fi

echo "Installing AgentRoom ${VERSION} (${OS}/${ARCH})..."

# ---- Determine install path ----
if [ "$(id -u)" = "0" ]; then
  INSTALL_BIN="/usr/local/bin"
  DATA_DIR="/var/lib/agentroom"
  AS_ROOT=1
else
  INSTALL_BIN="$HOME/.local/bin"
  DATA_DIR="$HOME/.agentroom"
  AS_ROOT=0
  mkdir -p "$INSTALL_BIN"
fi

mkdir -p "$DATA_DIR"

# ---- Download and verify ----
ARCHIVE="${BINARY}_${OS}_${ARCH}.tar.gz"
BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
TMP_DIR="$(mktemp -d)"

cleanup() { rm -rf "$TMP_DIR"; }
trap cleanup EXIT

echo "Downloading ${ARCHIVE}..."
curl -fsSL "${BASE_URL}/${ARCHIVE}" -o "${TMP_DIR}/${ARCHIVE}" || {
  echo "Download failed: ${BASE_URL}/${ARCHIVE}" >&2
  exit 1
}

echo "Downloading checksums..."
curl -fsSL "${BASE_URL}/checksums.txt" -o "${TMP_DIR}/checksums.txt" || {
  echo "Download failed: ${BASE_URL}/checksums.txt" >&2
  exit 1
}

echo "Verifying checksum..."
cd "$TMP_DIR"
if command -v sha256sum >/dev/null 2>&1; then
  grep "${ARCHIVE}" checksums.txt | sha256sum -c - || {
    echo "Checksum verification FAILED. Aborting." >&2
    exit 1
  }
elif command -v shasum >/dev/null 2>&1; then
  grep "${ARCHIVE}" checksums.txt | shasum -a 256 -c - || {
    echo "Checksum verification FAILED. Aborting." >&2
    exit 1
  }
else
  echo "Warning: no sha256sum or shasum found, skipping checksum verification" >&2
fi
cd - >/dev/null

echo "Extracting..."
tar -xzf "${TMP_DIR}/${ARCHIVE}" -C "$TMP_DIR"
chmod +x "${TMP_DIR}/${BINARY}"

# ---- Install binary ----
echo "Installing to ${INSTALL_BIN}/${BINARY}..."
cp "${TMP_DIR}/${BINARY}" "${INSTALL_BIN}/${BINARY}"

# ---- Install uninstaller ----
UNINSTALL_SCRIPT="${INSTALL_BIN}/agentroom-uninstall"
cat > "$UNINSTALL_SCRIPT" << UNINSTALL_EOF
#!/bin/sh
set -e
PURGE=0
for arg in "\$@"; do
  [ "\$arg" = "--purge" ] && PURGE=1
done
echo "Removing AgentRoom binary..."
rm -f "${INSTALL_BIN}/${BINARY}"
rm -f "${UNINSTALL_SCRIPT}"
if [ "\$PURGE" = "1" ]; then
  echo "Removing data directory ${DATA_DIR}..."
  rm -rf "${DATA_DIR}"
fi
if [ -f /etc/systemd/system/agentroom.service ]; then
  systemctl disable --now agentroom 2>/dev/null || true
  rm -f /etc/systemd/system/agentroom.service
  systemctl daemon-reload 2>/dev/null || true
fi
echo "AgentRoom uninstalled."
UNINSTALL_EOF
chmod +x "$UNINSTALL_SCRIPT"

# ---- systemd (root + Linux + systemd) ----
SYSTEMD_INSTALLED=0
if [ "$AS_ROOT" = "1" ] && [ "$OS" = "linux" ] && command -v systemctl >/dev/null 2>&1; then
  # Create agentroom user if missing
  if ! id agentroom >/dev/null 2>&1; then
    useradd -r -s /sbin/nologin -d "$DATA_DIR" agentroom 2>/dev/null || true
  fi
  chown -R agentroom:agentroom "$DATA_DIR" 2>/dev/null || true

  ADMIN_PASSWORD_LINE=""
  if [ -n "$AGENTROOM_ADMIN_PASSWORD" ]; then
    ADMIN_PASSWORD_LINE="Environment=AGENTROOM_ADMIN_PASSWORD=${AGENTROOM_ADMIN_PASSWORD}"
  fi

  cat > /etc/systemd/system/agentroom.service << SERVICE_EOF
[Unit]
Description=AgentRoom — shared inbox for AI coding agents
After=network.target

[Service]
ExecStart=${INSTALL_BIN}/${BINARY}
Environment=AGENTROOM_DB=${DATA_DIR}/agentroom.db
${ADMIN_PASSWORD_LINE}
Restart=on-failure
User=agentroom
WorkingDirectory=${DATA_DIR}

[Install]
WantedBy=multi-user.target
SERVICE_EOF

  systemctl daemon-reload
  systemctl enable agentroom
  systemctl restart agentroom
  SYSTEMD_INSTALLED=1
fi

# ---- Success banner ----
echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║       AgentRoom ${VERSION} installed successfully!              ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""
echo "  Binary: ${INSTALL_BIN}/${BINARY}"
echo "  Data:   ${DATA_DIR}"
echo ""

# ---- Run setup wizard ----
# Skip wizard if we just installed a systemd service (already configured above)
if [ "$SYSTEMD_INSTALLED" = "0" ]; then
  echo "  Running setup wizard..."
  echo ""
  "${INSTALL_BIN}/${BINARY}" init
else
  echo "  Service: systemctl status agentroom"
  echo "  URL:     http://$(hostname -I 2>/dev/null | awk '{print $1}' || echo 'localhost'):${DEFAULT_PORT}"
  echo ""
  echo "  Admin password: check the service logs:"
  echo "    journalctl -u agentroom | grep 'admin password'"
  echo ""
  echo "  Re-run setup:  agentroom init"
fi

echo ""
echo "  Uninstall: agentroom-uninstall [--purge]"
echo ""
