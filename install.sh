#!/usr/bin/env sh
# install.sh — install redash from GitHub releases.
#
# Usage:
#   curl -sSfL https://raw.githubusercontent.com/leroy/redash/main/install.sh | sh
#
# Environment variables:
#   VERSION      Tag to install (default: latest). Example: VERSION=v1.0.1
#   INSTALL_DIR  Destination directory (default: /usr/local/bin; falls
#                back to sudo if not writable). Example: INSTALL_DIR=$HOME/.local/bin
#   REPO         Override the source repo (default: leroy/redash).
#
# Supports macOS (darwin) and Linux on amd64/arm64. Windows users should
# download the zip from the Releases page manually.

set -eu

REPO="${REPO:-leroy/redash}"
BIN_NAME="redash"
VERSION="${VERSION:-latest}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

log() { printf 'redash install: %s\n' "$*" >&2; }
fatal() { printf 'redash install: %s\n' "$*" >&2; exit 1; }

# --- detect OS / architecture -----------------------------------------
uname_s=$(uname -s)
uname_m=$(uname -m)

case "$uname_s" in
  Darwin) os=darwin ;;
  Linux)  os=linux ;;
  *) fatal "unsupported OS '$uname_s'. Supported: Darwin, Linux. Download the archive manually from https://github.com/$REPO/releases" ;;
esac

case "$uname_m" in
  arm64|aarch64) arch=arm64 ;;
  x86_64|amd64)  arch=x86_64 ;;
  *) fatal "unsupported architecture '$uname_m'. Supported: x86_64, arm64. Download the archive manually from https://github.com/$REPO/releases" ;;
esac

# --- resolve version --------------------------------------------------
if [ "$VERSION" = "latest" ]; then
  api_url="https://api.github.com/repos/$REPO/releases/latest"
  VERSION=$(curl -sSfL "$api_url" \
    | awk -F'"' '/"tag_name":/ { print $4; exit }')
  if [ -z "$VERSION" ]; then
    fatal "could not resolve latest version from $api_url. Pass VERSION=vX.Y.Z explicitly, or check that the repo has published releases."
  fi
fi

version_num=${VERSION#v}
archive="${BIN_NAME}_${version_num}_${os}_${arch}.tar.gz"
archive_url="https://github.com/$REPO/releases/download/$VERSION/$archive"
checksums_url="https://github.com/$REPO/releases/download/$VERSION/checksums.txt"

log "installing $BIN_NAME $VERSION ($os/$arch) to $INSTALL_DIR"

# --- download ---------------------------------------------------------
tmp=$(mktemp -d 2>/dev/null || mktemp -d -t redash)
trap 'rm -rf "$tmp"' EXIT

log "downloading $archive_url"
if ! curl -sSfL "$archive_url" -o "$tmp/$archive"; then
  fatal "download failed. Tag $VERSION may not exist, or $archive may not be published for $os/$arch. See https://github.com/$REPO/releases/tag/$VERSION"
fi

# --- verify checksum --------------------------------------------------
log "verifying sha256"
if ! curl -sSfL "$checksums_url" -o "$tmp/checksums.txt"; then
  fatal "could not fetch $checksums_url. Refusing to install without checksum verification."
fi

expected=$(awk -v f="$archive" '$2 == f { print $1; exit }' "$tmp/checksums.txt")
if [ -z "$expected" ]; then
  fatal "'$archive' is not listed in $checksums_url. The release may be malformed; refusing to install."
fi

if command -v sha256sum >/dev/null 2>&1; then
  actual=$(sha256sum "$tmp/$archive" | awk '{print $1}')
elif command -v shasum >/dev/null 2>&1; then
  actual=$(shasum -a 256 "$tmp/$archive" | awk '{print $1}')
else
  fatal "neither sha256sum nor shasum is installed. Install coreutils (Linux) or Perl/shasum (macOS) and retry."
fi

if [ "$expected" != "$actual" ]; then
  fatal "sha256 mismatch for $archive (expected $expected, got $actual). Not installing. The downloaded file has been removed."
fi

# --- extract + install ------------------------------------------------
log "extracting"
tar -xzf "$tmp/$archive" -C "$tmp"

if [ ! -f "$tmp/$BIN_NAME" ]; then
  fatal "extracted archive does not contain '$BIN_NAME' at the top level. Archive layout may have changed; open an issue at https://github.com/$REPO/issues"
fi

mkdir -p "$INSTALL_DIR" 2>/dev/null || true
if [ -w "$INSTALL_DIR" ]; then
  cp "$tmp/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
elif command -v sudo >/dev/null 2>&1; then
  log "$INSTALL_DIR is not writable; using sudo"
  sudo cp "$tmp/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
else
  fatal "$INSTALL_DIR is not writable and sudo is not available. Set INSTALL_DIR to a writable path (e.g. INSTALL_DIR=\$HOME/.local/bin) and retry."
fi

log "installed $BIN_NAME $VERSION to $INSTALL_DIR/$BIN_NAME"
"$INSTALL_DIR/$BIN_NAME" version
