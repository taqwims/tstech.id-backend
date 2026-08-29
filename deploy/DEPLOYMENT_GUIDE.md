# Panduan Lengkap Deployment TsTech / Kotban

Panduan ini menjelaskan konfigurasi dan alur deployment arsitektur:
- **Backend (Go / Echo)**: Deploy di **VPS (Tanpa Docker)** menggunakan **Systemd**, **Nginx Reverse Proxy**, dan **Let's Encrypt SSL**.
- **Frontend (Next.js)**: Deploy di **Vercel**.
- **Database**: Cloud PostgreSQL (Neon.tech / Supabase) atau PostgreSQL / SQLite di VPS.

---

## 🏗️ Ringkasan Arsitektur

```mermaid
graph LR
    User[Pengunjung / Klien] -->|HTTPS: tstech.id| Vercel[Frontend - Vercel Next.js]
    User -->|HTTPS: api.tstech.id| Nginx[Nginx Reverse Proxy di VPS]
    Vercel -->|API Requests / SSR| Nginx
    Nginx -->|Port 8080| GoApp[Go Backend Binary Systemd]
    GoApp -->|Database Connection| DB[(Neon PostgreSQL / Cloud DB)]
    GoApp -->|Object Storage| R2[Cloudflare R2 Bucket]
```

---

## 📦 BAGIAN 1: Deploy Backend di VPS (Tanpa Docker)

Karena VPS berkapasitas kecil (misal 512MB / 1GB RAM), Go binary sangat ideal karena hanya membutuhkan memori RAM ~20-40 MB.

### 1.1 Persiapan VPS (Ubuntu / Debian)

Login ke VPS via SSH:
```bash
ssh root@IP_VPS_ANDA
```

Jalankan update sistem dan instalasi dependensi dasar:
```bash
sudo apt update && sudo apt upgrade -y
sudo apt install -y git curl wget build-essential nginx certbot python3-certbot-nginx ufw
```

*(Opsional tetapi Sangat Dianjurkan)* **Buat Swap Memory 1GB/2GB** agar VPS tidak kehabisan RAM saat build/running:
```bash
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

### 1.2 Install Golang di VPS

Unduh dan install Go versi terbaru (1.24+ / 1.25):
```bash
wget https://go.dev/dl/go1.24.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.24.0.linux-amd64.tar.gz
rm go1.24.0.linux-amd64.tar.gz

# Tambahkan ke PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
go version
```

### 1.3 Clone / Upload Project ke VPS

Buat direktori kerja di `/var/www/kotban`:
```bash
sudo mkdir -p /var/www/kotban
sudo chown -R $USER:$USER /var/www/kotban
cd /var/www/kotban

# Clone repository Anda:
git clone <URL_REPOSITORY_GIT_ANDA> .
```

### 1.4 Konfigurasi Environment Backend (`.env`)

Buat file `.env` di direktori `/var/www/kotban/backend/.env`:
```bash
cd /var/www/kotban/backend
nano .env
```

Contoh isi `.env` untuk VPS produksi:
```env
# 1. Database (Gunakan Neon.tech / Cloud PostgreSQL agar hemat RAM VPS)
DATABASE_URL="postgres://user:password@ep-xyz.ap-southeast-1.aws.neon.tech/neondb?sslmode=require"

# 2. Server API
API_ENV=production
PORT=8080

# 3. Cloudflare R2 Storage
R2_ACCOUNT_ID=e4a8102b7723e65d0d0659a2c8e609c6
R2_ACCESS_KEY_ID=cc07ebc20503151d26a32dc66f711a90
R2_SECRET_ACCESS_KEY=1d9ccecb4a8e43ad639ca08e20082d9e928ff5dae0ae65263efabeb38559fa9b
R2_BUCKET_NAME=sucommerce
R2_PUBLIC_URL=https://pub-88517502438642168ebfecf31829e56b.r2.dev

# 4. Authentication (JWT) & Admin Seed
JWT_SECRET=buat-string-rahasia-panjang-dan-acak-minimal-32-karakter
JWT_EXPIRE_HOURS=24
JWT_REFRESH_EXPIRE_HOURS=168
ADMIN_EMAIL=admin@tstech.id
ADMIN_PASSWORD=PasswordAdminAman123!
ADMIN_NAME=Admin TsTech

# 5. Frontend & App URL (Domain Vercel / Domain Utama)
APP_URL=https://tstech.id
APP_NAME=TsTech
WHATSAPP_NUMBER=628812209692

# 6. Payment Gateway
IPAYMU_VA=
IPAYMU_API_KEY=
IPAYMU_IS_PRODUCTION=true
IPAYMU_RETURN_URL=https://tstech.id/pemesanan/sukses
IPAYMU_CANCEL_URL=https://tstech.id/pemesanan/batal
IPAYMU_NOTIFY_URL=https://api.tstech.id/api/payments/notify

# 7. SMTP Email
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=05taqwim@gmail.com
SMTP_PASSWORD="app-password-gmail-anda"
SMTP_FROM_NAME=TsTech
SMTP_FROM_EMAIL=noreply@tstech.id
```

### 1.5 Compile Go Binary & Pasang Systemd Service

Salin file systemd service yang telah disediakan:
```bash
sudo cp /var/www/kotban/backend/deploy/systemd/backend.service /etc/systemd/system/backend.service

# Compile binary pertama kali
cd /var/www/kotban/backend
CGO_ENABLED=0 go build -ldflags="-s -w" -o server_bin ./cmd/server
chmod +x server_bin

# Reload daemon dan start service
sudo systemctl daemon-reload
sudo systemctl enable backend.service
sudo systemctl start backend.service

# Cek status service
sudo systemctl status backend.service
```

### 1.6 Konfigurasi Nginx & SSL Certbot

1. Salin konfigurasi Nginx:
```bash
sudo cp /var/www/kotban/backend/deploy/nginx/backend-api.conf /etc/nginx/sites-available/api.tstech.id
```

2. Edit nama domain sesuai domain/subdomain API Anda (`api.tstech.id`):
```bash
sudo nano /etc/nginx/sites-available/api.tstech.id
```
*(Ganti `api.yourdomain.com` dengan subdomain API Anda, misal `api.tstech.id`)*

3. Aktifkan konfigurasi Nginx dan restart:
```bash
sudo ln -s /etc/nginx/sites-available/api.tstech.id /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

4. Pasang SSL Gratis dengan Let's Encrypt Certbot:
*(Pastikan DNS A record subdomain sudah mengarah ke IP VPS sebelum menjalankan ini)*
```bash
sudo certbot --nginx -d api.tstech.id
```

5. Setup Firewall UFW:
```bash
sudo ufw allow 'OpenSSH'
sudo ufw allow 'Nginx Full'
sudo ufw enable
```

---

## ⚡ BAGIAN 2: Deploy Frontend di Vercel

### 2.1 Import Project di Vercel Dashboard

1. Buka [vercel.com](https://vercel.com) dan login dengan akun GitHub/GitLab Anda.
2. Klik **"Add New..."** -> **"Project"**.
3. Pilih repository `kotban.com` / `tstech` / `frontend`.
4. Pada form konfigurasi:
   - **Framework Preset**: `Next.js`
   - **Root Directory**: Klik `Edit` dan pilih folder **`frontend`** (atau `.` jika repo frontend terpisah)
   - **Build Command**: `next build` (default)
   - **Output Directory**: `.next` (default)
   - **Install Command**: `npm install` (default)

### 2.2 Atur Environment Variables di Vercel

Pada bagian **Environment Variables**, tambahkan:

| Key | Value Contoh | Deskripsi |
|---|---|---|
| `NEXT_PUBLIC_API_URL` | `https://api.tstech.id` | URL endpoint backend Go di VPS |
| `NEXT_PUBLIC_SITE_URL` | `https://tstech.id` | Domain frontend Anda |
| `NEXT_PUBLIC_WA_NUMBER` | `628812209692` | Nomor WhatsApp CS / Admin |

5. Klik tombol **Deploy**.

---

## 🌐 BAGIAN 3: Pengaturan DNS (Domain & Subdomain)

Di panel domain provider Anda (Cloudflare, Niagahoster, Namecheap, DomaiNesia, dll.), tambahkan DNS Records berikut:

| Tipe | Nama (Host) | Target / Value | Keterangan |
|---|---|---|---|
| **A** | `api` | `IP_VPS_ANDA` | Mengarahkan API Backend ke VPS |
| **CNAME** | `@` / `tstech.id` | `cname.vercel-dns.com` | Mengarahkan Frontend ke Vercel |
| **CNAME** | `www` | `cname.vercel-dns.com` | Mengarahkan www ke Vercel |

*(Jika menggunakan Cloudflare, untuk subdomain `api` matikan Proxy / ikon awan jadi Grey-cloud / DNS Only agar Certbot SSL di VPS berjalan mulus)*.

---

## 🔄 BAGIAN 4: Cara Update / Deploy Ulang Kode Baru

### Update Backend di VPS:
Cukup jalankan script yang sudah kami siapkan:
```bash
cd /var/www/kotban/backend/deploy
./deploy-vps.sh
```

### Update Frontend di Vercel:
Frontend akan otomatis ter-deploy setiap kali Anda melakukan `git push` ke branch utama di GitHub!
