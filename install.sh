#!/usr/bin/env bash
set -e

echo "========================================================================"
echo " RAWOPS v1.0.0 INSTALLER | BARE-METAL CONTROL PLANE (x86_64)"
echo "========================================================================"

ARCH=$(uname -m)
if [ "$ARCH" != "x86_64" ]; then
    echo "[ERR] Unsupported CPU architecture: $ARCH. RawOps requires x86_64."
    exit 1
fi

if [ "$(id -u)" -ne 0 ]; then
    echo "[ERR] Root privileges required. Run with sudo or as root."
    exit 1
fi

if [ -f "./rawops-linux-amd64" ]; then
    echo "[*] Installing local binary to /usr/local/bin/rawops..."
    install -m 0755 ./rawops-linux-amd64 /usr/local/bin/rawops
else
    URL="https://github.com"
    echo "[*] Downloading latest x86_64 release binary..."
    curl -fsSL "$URL" -o /usr/local/bin/rawops
    chmod 0755 /usr/local/bin/rawops
fi

mkdir -p /etc/rawops /var/backups/rawops
echo "[OK] RawOps v1.0.0 installed. Execute 'rawops status' to verify."
