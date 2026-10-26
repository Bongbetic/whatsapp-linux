#!/bin/sh
# Build the XBPS package for whatsapp-linux (plan §16, WL-7.1).
# Usage: bash tools/build-xbps.sh [arch]
# Env: VERSION file == git tag whatsapp-linux-vX.Y.Z (workflow fails
# on mismatch). Output lands in dist/.
set -eu

ARCH="${1:-$(xbps-uhelper arch 2>/dev/null || uname -m)}"
V="$(cat VERSION)"
# XBPS versions cannot carry '-': 0.0.0-dev -> 0.0.0.dev (test builds).
V="$(printf '%s' "$V" | tr '-' '.')"
REV=1
PKG="whatsapp-linux-${V}_${REV}"

rm -rf dist stage
mkdir -p dist stage

echo "==> build binary ($V)"
CGO_ENABLED=1 go build -tags sqlite_fts5 -trimpath \
  -ldflags "-s -w -X main.version=$V" \
  -o stage/whatsapp-linux ./cmd/whatsapp-linux

echo "==> stage tree"
mkdir -p stage/usr/bin stage/usr/share/applications \
  stage/usr/share/metainfo stage/usr/share/icons/hicolor/48x48/apps \
  stage/usr/share/icons/hicolor/scalable/apps \
  stage/usr/share/glib-2.0/schemas \
  stage/usr/share/sounds/whatsapp-linux \
  stage/usr/share/whatsapp-linux \
  stage/usr/share/doc/whatsapp-linux
mv stage/whatsapp-linux stage/usr/bin/whatsapp-linux
cp data/style.css stage/usr/share/whatsapp-linux/
cp data/com.bongbetic.WhatsAppLinux.desktop stage/usr/share/applications/
cp data/com.bongbetic.WhatsAppLinux.metainfo.xml stage/usr/share/metainfo/
cp data/icons/hicolor/48x48/apps/com.bongbetic.WhatsAppLinux.png \
  stage/usr/share/icons/hicolor/48x48/apps/
cp data/icons/hicolor/scalable/apps/com.bongbetic.WhatsAppLinux.svg \
  stage/usr/share/icons/hicolor/scalable/apps/
cp data/com.bongbetic.WhatsAppLinux.gschema.xml stage/usr/share/glib-2.0/schemas/
cp data/sounds/*.wav data/sounds/CREDITS stage/usr/share/sounds/whatsapp-linux/
cp LICENSE NOTICE README.md stage/usr/share/doc/whatsapp-linux/

echo "==> validate"
glib-compile-schemas --strict --dry-run stage/usr/share/glib-2.0/schemas/
desktop-file-validate stage/usr/share/applications/com.bongbetic.WhatsAppLinux.desktop

echo "==> xbps-create"
xbps-create -A "$ARCH" -n "$PKG" \
  -s "Unofficial WhatsApp client for Linux" \
  --license "MIT, MPL-2.0, GPL-3.0-only" \
  --dependencies "gtk4>=4.16 gstreamer1>=1.28 gst-plugins-base1>=1.28 gst-plugins-good1>=1.28 gst-plugins-bad1>=1.28 pipewire>=1.0 sqlite>=3.0 libwebp>=1.0 libgirepository>=1.80 xdg-desktop-portal>=1.0 hicolor-icon-theme>=0.17 desktop-file-utils>=0.23" \
  stage/
mv ./*.xbps dist/ 2>/dev/null || true
ls -la dist/
echo "==> done: dist/$PKG.$ARCH.xbps"
