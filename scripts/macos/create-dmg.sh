#!/bin/bash
# Create macOS DMG installer for OpenPrintHub

set -e

# Configuration
APP_NAME="OpenPrintHub"
VERSION="${VERSION:-0.1.0}"
BINARY_PATH="${BINARY_PATH:-./oph}"
OUTPUT_DIR="${OUTPUT_DIR:-.}"
ARCH="${ARCH:-arm64}"

# Derived names
DMG_NAME="${APP_NAME}-${VERSION}-${ARCH}.dmg"
APP_BUNDLE="${APP_NAME}.app"
VOLUME_NAME="${APP_NAME} ${VERSION}"

echo "Creating macOS DMG for ${APP_NAME} v${VERSION} (${ARCH})"

# Create temporary directory
TEMP_DIR=$(mktemp -d)
trap "rm -rf ${TEMP_DIR}" EXIT

# Create app bundle structure
APP_DIR="${TEMP_DIR}/${APP_BUNDLE}"
mkdir -p "${APP_DIR}/Contents/MacOS"
mkdir -p "${APP_DIR}/Contents/Resources"

# Copy binary
cp "${BINARY_PATH}" "${APP_DIR}/Contents/MacOS/oph"
chmod +x "${APP_DIR}/Contents/MacOS/oph"

# Create Info.plist
cat > "${APP_DIR}/Contents/Info.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>CFBundleDevelopmentRegion</key>
    <string>en</string>
    <key>CFBundleExecutable</key>
    <string>oph</string>
    <key>CFBundleIdentifier</key>
    <string>com.openprintub.oph</string>
    <key>CFBundleInfoDictionaryVersion</key>
    <string>6.0</string>
    <key>CFBundleName</key>
    <string>${APP_NAME}</string>
    <key>CFBundlePackageType</key>
    <string>APPL</string>
    <key>CFBundleShortVersionString</key>
    <string>${VERSION}</string>
    <key>CFBundleVersion</key>
    <string>${VERSION}</string>
    <key>LSMinimumSystemVersion</key>
    <string>10.15</string>
    <key>NSHighResolutionCapable</key>
    <true/>
    <key>LSUIElement</key>
    <true/>
</dict>
</plist>
EOF

# Create a simple icon if not exists (placeholder)
# In production, you'd want to include a proper .icns file

# Create DMG staging directory
DMG_STAGING="${TEMP_DIR}/dmg-staging"
mkdir -p "${DMG_STAGING}"

# Copy app bundle to staging
cp -R "${APP_DIR}" "${DMG_STAGING}/"

# Create Applications symlink
ln -s /Applications "${DMG_STAGING}/Applications"

# Create README in DMG
cat > "${DMG_STAGING}/README.txt" << EOF
OpenPrintHub v${VERSION}

Installation:
1. Drag OpenPrintHub.app to the Applications folder
2. Open Terminal and run: /Applications/OpenPrintHub.app/Contents/MacOS/oph
3. The service will start on http://localhost:16800

For more information, visit: https://github.com/medpath1024/OpenPrintHub

To run as a background service, you can create a LaunchAgent:
  cp /Applications/OpenPrintHub.app/Contents/Resources/com.openprintub.oph.plist ~/Library/LaunchAgents/
  launchctl load ~/Library/LaunchAgents/com.openprintub.oph.plist
EOF

# Create LaunchAgent plist
mkdir -p "${APP_DIR}/Contents/Resources"
cat > "${APP_DIR}/Contents/Resources/com.openprintub.oph.plist" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.openprintub.oph</string>
    <key>ProgramArguments</key>
    <array>
        <string>/Applications/OpenPrintHub.app/Contents/MacOS/oph</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/tmp/openprintub.log</string>
    <key>StandardErrorPath</key>
    <string>/tmp/openprintub.error.log</string>
</dict>
</plist>
EOF

# Re-copy app bundle with LaunchAgent
rm -rf "${DMG_STAGING}/${APP_BUNDLE}"
cp -R "${APP_DIR}" "${DMG_STAGING}/"

# Create DMG
echo "Creating DMG..."
hdiutil create -volname "${VOLUME_NAME}" \
    -srcfolder "${DMG_STAGING}" \
    -ov -format UDZO \
    "${OUTPUT_DIR}/${DMG_NAME}"

echo "Created: ${OUTPUT_DIR}/${DMG_NAME}"
