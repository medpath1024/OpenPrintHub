#!/bin/bash
# Create macOS PKG installer for OpenPrintHub as a launchd service

set -euo pipefail

APP_NAME="OpenPrintHub"
IDENTIFIER="com.openprinthub.oph"
VERSION="${VERSION:-0.1.5}"
BINARY_PATH="${BINARY_PATH:-./oph}"
OUTPUT_DIR="${OUTPUT_DIR:-.}"
ARCH="${ARCH:-arm64}"
PORT="${PORT:-16800}"
WEB_PORT="${WEB_PORT:-16801}"

PKG_NAME="${APP_NAME}-${VERSION}-${ARCH}.pkg"

if [ ! -f "${BINARY_PATH}" ]; then
    echo "error: binary not found: ${BINARY_PATH}" >&2
    exit 1
fi

echo "Creating macOS PKG for ${APP_NAME} v${VERSION} (${ARCH})"

TEMP_DIR=$(mktemp -d)
trap "rm -rf ${TEMP_DIR}" EXIT

ROOT_DIR="${TEMP_DIR}/root"
SCRIPTS_DIR="${TEMP_DIR}/scripts"

mkdir -p "${ROOT_DIR}/usr/local/bin"
mkdir -p "${ROOT_DIR}/Library/LaunchDaemons"
mkdir -p "${SCRIPTS_DIR}"

cp "${BINARY_PATH}" "${ROOT_DIR}/usr/local/bin/oph"
chmod 755 "${ROOT_DIR}/usr/local/bin/oph"

cat > "${ROOT_DIR}/Library/LaunchDaemons/${IDENTIFIER}.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>${IDENTIFIER}</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/oph</string>
        <string>-port</string>
        <string>${PORT}</string>
        <string>-web-port</string>
        <string>${WEB_PORT}</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/var/log/openprinthub.log</string>
    <key>StandardErrorPath</key>
    <string>/var/log/openprinthub.error.log</string>
</dict>
</plist>
EOF

cat > "${SCRIPTS_DIR}/preinstall" << 'EOF'
#!/bin/bash
set -e

# Unload both new and legacy labels before upgrade
launchctl bootout system/com.openprinthub.oph >/dev/null 2>&1 || true
launchctl bootout system/com.openprintub.oph >/dev/null 2>&1 || true
exit 0
EOF

cat > "${SCRIPTS_DIR}/postinstall" << 'EOF'
#!/bin/bash
set -e

PLIST_PATH="/Library/LaunchDaemons/com.openprinthub.oph.plist"
BIN_PATH="/usr/local/bin/oph"
LABEL="com.openprinthub.oph"

chown root:wheel "${PLIST_PATH}" "${BIN_PATH}"
chmod 644 "${PLIST_PATH}"
chmod 755 "${BIN_PATH}"

launchctl bootstrap system "${PLIST_PATH}" >/dev/null 2>&1 || true
launchctl enable "system/${LABEL}" >/dev/null 2>&1 || true
launchctl kickstart -k "system/${LABEL}" >/dev/null 2>&1 || true
exit 0
EOF

chmod 755 "${SCRIPTS_DIR}/preinstall" "${SCRIPTS_DIR}/postinstall"

mkdir -p "${OUTPUT_DIR}"
pkgbuild \
    --root "${ROOT_DIR}" \
    --scripts "${SCRIPTS_DIR}" \
    --identifier "${IDENTIFIER}" \
    --version "${VERSION}" \
    --install-location "/" \
    "${OUTPUT_DIR}/${PKG_NAME}"

echo "Created: ${OUTPUT_DIR}/${PKG_NAME}"
