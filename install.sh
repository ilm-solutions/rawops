#!/usr/bin/env bash
set -Eeuo pipefail

PREFIX="${PREFIX:-/usr/local}"
BINARY="./rawops-linux-amd64"

echo "RAWOPS installer"

if [[ "$(uname -s)" != "Linux" ]]; then
  echo "[ERR] RAWOPS currently supports Linux hosts only." >&2
  exit 1
fi
if [[ "$(uname -m)" != "x86_64" ]]; then
  echo "[ERR] This bundled binary targets x86_64; build for your architecture." >&2
  exit 1
fi
if [[ "$(id -u)" -ne 0 ]]; then
  echo "[ERR] Root privileges required. Run with sudo." >&2
  exit 1
fi
if [[ ! -f "$BINARY" ]]; then
  echo "[ERR] Local release binary not found: $BINARY" >&2
  echo "Build it with: go build -trimpath -o rawops-linux-amd64 ." >&2
  exit 1
fi

install -d -m 0755 "$PREFIX/bin"
install -m 0755 "$BINARY" "$PREFIX/bin/rawops"
"$PREFIX/bin/rawops" --help >/dev/null
echo "[OK] Installed $PREFIX/bin/rawops"
