#!/usr/bin/env bash
set -e

# ==============================================================================
# Script Otomatisasi Deploy Backend Go di VPS (Tanpa Docker)
# Lokasi: /var/www/kotban/backend/deploy/deploy-vps.sh
# ==============================================================================

APP_DIR="/var/www/kotban/backend"
SERVICE_NAME="backend.service"

echo "=========================================="
echo "🚀 Memulai Deployment Backend Go di VPS"
echo "=========================================="

cd "$APP_DIR"

# 1. Update source code dari Git (jika menggunakan git)
if [ -d ".git" ] || [ -d "../.git" ]; then
    echo "📥 Mengambil kode terbaru dari Git..."
    git pull origin main || git pull origin master || echo "⚠️ Git pull dilewati."
fi

# 2. Build Binary Go dengan optimasi ukuran
echo "🔨 Mengompilasi Go Binary (server_bin)..."
CGO_ENABLED=0 go build -ldflags="-s -w" -o server_bin ./cmd/server

# Pastikan file binary memiliki permission eksekusi
chmod +x server_bin

# 3. Restart Systemd Service
echo "🔄 Me-restart Systemd Service ($SERVICE_NAME)..."
systemctl restart "$SERVICE_NAME"

# Tunggu sejenak agar server inisialisasi
sleep 2

# 4. Validasi Status Service & Health Check
echo "🔍 Memeriksa Status Layanan..."
if systemctl is-active --quiet "$SERVICE_NAME"; then
    echo "✅ Service aktif dan berjalan!"
    echo "🌐 Melakukan health check..."
    curl -s http://127.0.0.1:8080/api/health || true
    echo ""
    echo "=========================================="
    echo "🎉 Deployment Backend Berhasil!"
    echo "=========================================="
else
    echo "❌ Service gagal berjalan! Menampilkan log error:"
    journalctl -u "$SERVICE_NAME" -n 20 --no-pager
    exit 1
fi
