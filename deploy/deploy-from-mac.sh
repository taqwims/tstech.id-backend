#!/usr/bin/env bash
set -e

# ==============================================================================
# Script Build di Mac & Otomatis Upload ke VPS (Tanpa Perlu Go di VPS)
# Jalankan dari Mac: ./deploy/deploy-from-mac.sh <USER@IP_VPS>
# Contoh: ./deploy/deploy-from-mac.sh root@123.45.67.89
# ==============================================================================

VPS_TARGET="$1"
ARCH="${2:-amd64}" # default: amd64 (atau arm64 jika VPS berbasis ARM)
REMOTE_DIR="/var/www/kotban/backend"

if [ -z "$VPS_TARGET" ]; then
    echo "❌ Error: Masukkan user dan IP VPS!"
    echo "Penggunaan: $0 <user@ip_vps> [amd64|arm64]"
    echo "Contoh:     $0 root@103.123.45.67"
    exit 1
fi

echo "=================================================="
echo "🔨 1. Mengompilasi Go Binary untuk Linux ($ARCH)..."
echo "=================================================="

cd "$(dirname "$0")/.."
CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" go build -ldflags="-s -w" -o server_bin ./cmd/server

echo "✅ Binary berhasil dibuat: $(pwd)/server_bin ($(du -h server_bin | cut -f1))"

echo "=================================================="
echo "📦 2. Memastikan direktori target ada di VPS..."
echo "=================================================="

ssh "$VPS_TARGET" "mkdir -p $REMOTE_DIR"

echo "=================================================="
echo "🚀 3. Mengupload Binary ke VPS ($VPS_TARGET)..."
echo "=================================================="

scp server_bin "$VPS_TARGET:$REMOTE_DIR/server_bin.new"

echo "=================================================="
echo "🔄 4. Me-replace binary dan restart service di VPS..."
echo "=================================================="

ssh "$VPS_TARGET" bash -c "'
    cd $REMOTE_DIR
    mv -f server_bin.new server_bin
    chmod +x server_bin
    if systemctl is-active --quiet backend.service; then
        systemctl restart backend.service
        echo \"✅ Service backend.service berhasil di-restart!\"
    else
        echo \"⚠️ Service backend.service belum aktif atau belum diinstall.\"
    fi
'"

echo "=================================================="
echo "🎉 Selesai! Binary Go berhasil dideploy ke VPS."
echo "=================================================="
