#!/usr/bin/env bash
set -e

echo "========================================================================"
echo " RAWOPS ENGINE INSTALLER | BARE-METAL CONTROL PLANE (x86_64)"
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

echo "[*] Installing rawops-linux-amd64 to /usr/local/bin/rawops..."
install -m 0755 rawops-linux-amd64 /usr/local/bin/rawops
mkdir -p /etc/rawops

echo "[OK] Installation complete. Execute 'rawops status' to initialize."
