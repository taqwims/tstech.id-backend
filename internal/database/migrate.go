package database

import (
	"log"
	"time"

	"github.com/kotban/backend/internal/config"
	"github.com/kotban/backend/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Migrate(db *gorm.DB, cfg *config.Config) {
	err := db.AutoMigrate(
		&model.Consultation{},
		&model.Order{},
		&model.Payment{},
		&model.Portfolio{},
		&model.Testimonial{},
		&model.Contact{},
		&model.User{},
		&model.Project{},
		&model.Milestone{},
		&model.ProjectComment{},
		&model.ProjectFile{},
		&model.Quotation{},
		&model.SiteContent{},
		&model.PaymentSetting{},
		&model.Article{},
	)
	if err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("✅ Database migrations completed")

	// Auto seed Admin User
	seedAdmin(db, cfg)

	// Auto seed Default Payment Settings
	seedDefaultPaymentSettings(db, cfg)

	// Auto seed Default Site Content
	seedDefaultContent(db)

	// Auto seed Default Portfolios & Testimonials
	seedPortfoliosAndTestimonials(db)

	// Auto seed Default SEO Articles
	seedArticles(db)
}

func seedAdmin(db *gorm.DB, cfg *config.Config) {
	var count int64
	db.Model(&model.User{}).Where("role = ?", "admin").Count(&count)
	if count == 0 {
		hashed, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("⚠️ Failed to hash admin password: %v", err)
			return
		}

		admin := model.User{
			Email:       cfg.AdminEmail,
			Password:    string(hashed),
			Name:        cfg.AdminName,
			Role:        "admin",
			Phone:       cfg.WhatsAppNumber,
			CompanyName: "Kotban.com",
			IsActive:    true,
		}

		if err := db.Create(&admin).Error; err != nil {
			log.Printf("⚠️ Failed to seed admin user: %v", err)
		} else {
			log.Printf("👑 Seeded default admin user: %s (Password: %s)", cfg.AdminEmail, cfg.AdminPassword)
		}
	}
}

func seedDefaultContent(db *gorm.DB) {
	var count int64
	db.Model(&model.SiteContent{}).Count(&count)
	if count > 0 {
		return
	}

	defaults := []model.SiteContent{
		{
			Key:   "hero_headline",
			Value: "Wujudkan Software Impian Bisnis Anda, Cepat & Profesional",
			Type:  "text",
			Group: "hero",
		},
		{
			Key:   "hero_subheadline",
			Value: "Jasa pembuatan website, aplikasi mobile, sistem informasi, dan software custom untuk bisnis Anda.",
			Type:  "text",
			Group: "hero",
		},
		{
			Key:   "hero_badge",
			Value: "Dipercaya 50+ Klien di Seluruh Indonesia",
			Type:  "text",
			Group: "hero",
		},
		{
			Key:   "about_company",
			Value: "Kotban.com adalah software house modern yang berfokus pada pengembangan solusi digital berkualitas tinggi, scalable, dan tepat waktu untuk berbagai kebutuhan bisnis.",
			Type:  "text",
			Group: "about",
		},
		{
			Key:   "contact_email",
			Value: "halo@kotban.com",
			Type:  "text",
			Group: "contact",
		},
		{
			Key:   "contact_phone",
			Value: "+62 812-3456-7890",
			Type:  "text",
			Group: "contact",
		},
		{
			Key:   "contact_address",
			Value: "Menara Prima Lt. 18, Mega Kuningan, Jakarta Selatan",
			Type:  "text",
			Group: "contact",
		},
		{
			Key:   "site_services",
			Value: `[{"id":1,"icon":"🌐","title":"Website Company Profile","desc":"Website profesional untuk meningkatkan kredibilitas bisnis Anda.","features":["Responsive design","CMS Admin panel","SEO optimization","Analytics integration"]},{"id":2,"icon":"🛒","title":"Website E-Commerce","desc":"Toko online lengkap dengan sistem pembayaran & manajemen produk.","features":["Katalog produk","Payment gateway iPaymu","Manajemen pesanan","Hitung ongkir otomatis"]},{"id":3,"icon":"📱","title":"Aplikasi Mobile (Android & iOS)","desc":"Aplikasi mobile modern native atau cross-platform berkualitas tinggi.","features":["Android & iOS","Push notification","Integrasi REST API","Publikasi Play Store"]},{"id":4,"icon":"⚙️","title":"Sistem Informasi & ERP Custom","desc":"Sistem custom untuk otomasi proses bisnis, operasional gudang, dan keuangan.","features":["Custom workflow","Multi-user & role","Dashboard laporan","Export data Excel/PDF"]},{"id":5,"icon":"🎨","title":"Desain UI/UX & Prototyping","desc":"Desain antarmuka modern, interaktif, dan user-centered di Figma.","features":["User research","Design system","Interactive prototype","Design tokens"]},{"id":6,"icon":"🔧","title":"Maintenance & Cloud Support","desc":"Dukungan teknis, backup rutin, dan pemeliharaan server berkala.","features":["Bug fixing","Security updates","Monitoring server","Backup rutin"]}]`,
			Type:  "json",
			Group: "services",
		},
		{
			Key:   "site_pricing_packages",
			Value: `[{"id":1,"name":"Basic","price":3500000,"price_display":"Rp 3.500.000","pages":"5 Halaman","design":"Template Premium","revisions":"1x Revisi","duration":"7-14 hari","support":"1 bulan","highlighted":false,"features":["5 Halaman Utama","Desain Template Premium","Responsive Mobile & Desktop","Domain & Hosting 1 Tahun","SSL Certificate","1x Revisi","Garansi & Support 1 Bulan"]},{"id":2,"name":"Professional","price":8000000,"price_display":"Rp 8.000.000","pages":"10+ Halaman","design":"Full Custom Design","revisions":"3x Revisi","duration":"14-30 hari","support":"3 bulan","highlighted":true,"features":["10+ Halaman / Fitur","Desain Full Custom Figma","CMS Admin Panel Lengkap","SEO Optimization & Analytics","Domain & Hosting Cloud 1 Tahun","SSL Certificate","3x Revisi Besar","Support Prioritas 3 Bulan"]},{"id":3,"name":"Enterprise","price":20000000,"price_display":"Rp 20.000.000","pages":"Unlimited","design":"Exclusive Tailored","revisions":"Unlimited*","duration":"30-60 hari","support":"6 bulan","highlighted":false,"features":["Kapasitas & Fitur Unlimited*","Arsitektur Microservices / Custom API","Integrasi Database & Payment Gateway","Multi-Level User & Permission Role","Deployment Cloud Server","Dokumentasi Lengkap & Handover Source Code","Garansi & Dedicated Support 6 Bulan"]}]`,
			Type:  "json",
			Group: "pricing",
		},
		{
			Key:   "site_workflow",
			Value: `[{"step":1,"title":"Konsultasi & Brief","desc":"Diskusi kebutuhan fitur, target audiens, dan tujuan bisnis Anda.","icon":"💬"},{"step":2,"title":"Proposal & Penawaran","desc":"Penerbitan surat penawaran harga transparan & timeline pengerjaan.","icon":"📋"},{"step":3,"title":"Desain UI/UX & Prototipe","desc":"Perancangan mockup visual interaktif di Figma sebelum coding.","icon":"🎨"},{"step":4,"title":"Development & Coding","desc":"Pengembangan sistem dengan standar clean code dan performa cepat.","icon":"💻"},{"step":5,"title":"Testing & QA","desc":"Pengujian menyeluruh, pengecekan keamanan, dan perbaikan revisi.","icon":"🧪"},{"step":6,"title":"Launching & Serah Terima","desc":"Deploy ke production server, serah terima kode, dan pelatihan.","icon":"🚀"}]`,
			Type:  "json",
			Group: "workflow",
		},
		{
			Key:   "site_faqs",
			Value: `[{"category":"Umum","items":[{"question":"Apa itu Kotban.com?","answer":"Kotban.com adalah software house profesional yang melayani jasa pembuatan website, aplikasi mobile (Android/iOS), sistem informasi custom (ERP/CRM/POS), UI/UX design, serta pemeliharaan software untuk berbagai skala bisnis."},{"question":"Layanan apa saja yang tersedia?","answer":"Kami menyediakan pembuatan Website Company Profile, Toko Online (E-Commerce), Aplikasi Mobile (Android/iOS), Sistem Informasi Custom (ERP, CRM, HRIS), Maintenance & Support, serta Jasa Desain UI/UX."}]},{"category":"Proses & Waktu Pengerjaan","items":[{"question":"Berapa lama waktu pengerjaan website/aplikasi?","answer":"Estimasi waktu pengerjaan bervariasi tergantung skala proyek. Paket Basic membutuhkan 7-14 hari kerja, Professional 14-30 hari kerja, dan Enterprise 30-60 hari kerja."},{"question":"Bagaimana proses revisi?","answer":"Revisi dilakukan pada tahap testing & QA. Jumlah revisi menyesuaikan paket yang dipilih (Basic: 1x, Professional: 3x, Enterprise: Unlimited sesuai ruang lingkup brief awal)."}]},{"category":"Pembayaran","items":[{"question":"Berapa besar DP yang dibutuhkan?","answer":"DP (Down Payment) standar adalah sebesar 50% dari total nilai proyek saat pesanan dibuat, dan pelunasan 50% sisanya dilakukan setelah testing selesai dan produk siap dilaunching."},{"question":"Metode pembayaran apa saja yang diterima?","answer":"Kami menerima pembayaran via iPaymu Payment Gateway yang mendukung Bank Transfer (Virtual Account), QRIS, E-Wallet, serta Kartu Kredit."}]},{"category":"Garansi & Hak Milik","items":[{"question":"Apakah klien mendapatkan source code dan hak milik penuh?","answer":"Ya, 100% hak cipta, source code, dan akses server akan diserahkan sepenuhnya kepada klien setelah pelunasan."},{"question":"Apakah ada garansi setelah selesai?","answer":"Kami memberikan garansi perbaikan bug dan error gratis selama 30 hingga 180 hari sesuai paket yang dipilih."}]}]`,
			Type:  "json",
			Group: "faq",
		},
		{
			Key:   "site_why_us",
			Value: `[{"icon":"👨‍💻","title":"Tim Berpengalaman","desc":"Senior developer dan UI/UX designer bersertifikasi dengan portofolio beragam."},{"icon":"💰","title":"Harga Transparan & Fleksibel","desc":"Biaya terstruktur tanpa biaya tersembunyi dengan termin pembayaran fleksibel."},{"icon":"🔄","title":"Garansi Kepuasan & Revisi","desc":"Jaminan pengerjaan sesuai brief dan revisi berkala di setiap tahapan."},{"icon":"🛡️","title":"Support & Maintenance Prioritas","desc":"Dukungan teknis responsif dan pemeliharaan server pasca-peluncuran."},{"icon":"⏰","title":"Komitmen Deadline Tepat Waktu","desc":"Manajemen timeline transparan yang dapat dipantau langsung dari dashboard klien."}]`,
			Type:  "json",
			Group: "about",
		},
	}

	for _, item := range defaults {
		var existing model.SiteContent
		if err := db.Where("`key` = ? OR \"key\" = ?", item.Key, item.Key).First(&existing).Error; err != nil {
			db.Create(&item)
		}
	}
	log.Println("📝 Ensured default site contents and CMS schemas")
}

func seedPortfoliosAndTestimonials(db *gorm.DB) {
	var pCount int64
	db.Model(&model.Portfolio{}).Count(&pCount)
	if pCount == 0 {
		samples := []model.Portfolio{
			{
				Slug:        "e-commerce-fashion-brand",
				Title:       "Platform E-Commerce Fashion Brand",
				Category:    "Website",
				ClientName:  "Zola Official",
				Industry:    "Retail & Fashion",
				Problem:     "Klien membutuhkan toko online dengan handling order ribuan transaksi per hari.",
				Solution:    "Kami membangun web store berbasis Next.js & Go microservice terintegrasi payment gateway.",
				TechStack:   "Next.js, Go, PostgreSQL, Redis, iPaymu",
				Result:      "Peningkatan transaksi 180% dan loading time di bawah 1 detik.",
				DemoURL:     "https://example.com",
				IsFeatured:  true,
				SortOrder:   1,
			},
			{
				Slug:        "pos-sistem-kasir-restoran",
				Title:       "Aplikasi Mobile POS & Manajemen Restoran",
				Category:    "Mobile App",
				ClientName:  "Kopi Nusantara Group",
				Industry:    "F&B",
				Problem:     "Pencatatan kasir manual antar 15 cabang rawan selisih stok.",
				Solution:    "Aplikasi tablet & mobile kasir tersinkronisasi realtime ke cloud dashboard.",
				TechStack:   "Flutter, Go, PostgreSQL, WebSocket",
				Result:      "Efisiensi order 40% lebih cepat dan laporan keuangan realtime.",
				DemoURL:     "https://example.com",
				IsFeatured:  true,
				SortOrder:   2,
			},
			{
				Slug:        "sistem-erp-manufaktur",
				Title:       "Sistem ERP Custom Pabrik & Gudang",
				Category:    "Sistem Informasi",
				ClientName:  "PT Surya Logistik",
				Industry:    "Manufaktur & Logistik",
				Problem:     "Manajemen inventori ribuan SKU yang tersebar di 4 gudang.",
				Solution:    "Sistem ERP berbasis web terintegrasi barcode scanner & auto-purchase order.",
				TechStack:   "React, Echo Go, PostgreSQL, Docker",
				Result:      "Reduksi human error hingga 95% dan audit stock opname realtime.",
				DemoURL:     "https://example.com",
				IsFeatured:  true,
				SortOrder:   3,
			},
		}
		for _, s := range samples {
			db.Create(&s)
		}
		log.Println("🎨 Seeded sample portfolios")
	}

	var tCount int64
	db.Model(&model.Testimonial{}).Count(&tCount)
	if tCount == 0 {
		testimoniList := []model.Testimonial{
			{
				ClientName:  "Ahmad Ridwan",
				CompanyName: "PT Maju Bersama",
				Rating:      5,
				Content:     "Kotban.com membantu kami membuat website e-commerce yang sangat profesional. Penjualan online kami meningkat 150% setelah launch!",
				IsActive:    true,
				SortOrder:   1,
			},
			{
				ClientName:  "Sari Dewi",
				CompanyName: "Klinik Sehat",
				Rating:      5,
				Content:     "Sistem informasi klinik yang dibuat sangat memudahkan operasional kami. Tim Kotban sangat responsif dan profesional.",
				IsActive:    true,
				SortOrder:   2,
			},
			{
				ClientName:  "Budi Santoso",
				CompanyName: "CV Jaya Abadi",
				Rating:      5,
				Content:     "Aplikasi mobile untuk sales tracking kami berjalan lancar. Revisi cepat dan support setelah selesai sangat membantu.",
				IsActive:    true,
				SortOrder:   3,
			},
		}
		for _, t := range testimoniList {
			db.Create(&t)
		}
		log.Println("⭐ Seeded sample testimonials")
	}
}

func seedDefaultPaymentSettings(db *gorm.DB, cfg *config.Config) {
	var count int64
	db.Model(&model.PaymentSetting{}).Count(&count)
	if count > 0 {
		return
	}

	defaults := []model.PaymentSetting{
		{Key: "mayar_enabled", Value: "true"},
		{Key: "mayar_api_key", Value: ""},
		{Key: "mayar_is_production", Value: "false"},
		{Key: "mayar_webhook_token", Value: ""},
		{Key: "ipaymu_enabled", Value: "true"},
		{Key: "ipaymu_va", Value: cfg.IpaymuVA},
		{Key: "ipaymu_api_key", Value: cfg.IpaymuAPIKey},
		{Key: "ipaymu_is_production", Value: "false"},
		{Key: "manual_transfer_enabled", Value: "true"},
		{Key: "manual_bank_accounts", Value: `[{"bank_name":"BCA","account_number":"8735098231","account_holder":"PT KOTBAN SOLUSI TEKNOLOGI","icon":"bca"},{"bank_name":"Bank Mandiri","account_number":"1370019827364","account_holder":"PT KOTBAN SOLUSI TEKNOLOGI","icon":"mandiri"},{"bank_name":"Bank Syariah Indonesia (BSI)","account_number":"7219082341","account_holder":"PT KOTBAN SOLUSI TEKNOLOGI","icon":"bsi"}]`},
		{Key: "manual_instructions", Value: "Silakan lakukan transfer tepat sesuai total nominal yang tertera ke salah satu rekening resmi di atas. Setelah transfer berhasil, harap unggah bukti transfer melalui halaman ini atau kirimkan konfirmasi via WhatsApp kami agar pesanan Anda dapat langsung diproses."},
		{Key: "manual_whatsapp", Value: cfg.WhatsAppNumber},
		{Key: "default_gateway", Value: "customer_choice"},
	}

	for _, s := range defaults {
		var existing model.PaymentSetting
		if err := db.Where("`key` = ? OR \"key\" = ?", s.Key, s.Key).First(&existing).Error; err != nil {
			db.Create(&s)
		}
	}
	log.Println("💳 Seeded default payment settings (Mayar.id, iPaymu, Manual Transfer)")
}

func seedArticles(db *gorm.DB) {
	var count int64
	db.Model(&model.Article{}).Count(&count)
	if count > 0 {
		return
	}

	now := time.Now()
	twoDaysAgo := now.Add(-48 * time.Hour)
	threeDaysAgo := now.Add(-72 * time.Hour)
	fiveDaysAgo := now.Add(-120 * time.Hour)
	oneWeekAgo := now.Add(-168 * time.Hour)

	articles := []model.Article{
		{
			Title:       "Panduan Lengkap SEO Website 2026: Strategi Ampuh Raih Peringkat #1 Google",
			Slug:        "panduan-lengkap-seo-website-2026",
			Excerpt:     "Pelajari kaidah SEO terbaru 2026 mulai dari Core Web Vitals, Search Intent, Structured Data JSON-LD, hingga optimasi AI Search untuk mendominasi peringkat pertama Google.",
			Category:    "SEO & Web Optimization",
			Tags:        "SEO, Google Search, Core Web Vitals, Web Optimization, Search Intent",
			AuthorName:  "Taqwim",
			AuthorAvatar: "https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=150&auto=format&fit=crop&q=80",
			AuthorRole:  "Lead SEO & Fullstack Architect",
			CoverImage:  "https://images.unsplash.com/photo-1460925895917-afdab827c52f?w=1200&auto=format&fit=crop&q=80",
			Status:      "published",
			IsFeatured:  true,
			IsTrending:  true,
			ViewsCount:  1420,
			ReadingTime: 7,
			MetaTitle:   "Panduan Lengkap SEO Website 2026: Strategi Ampuh Peringkat #1 | Kotban.com",
			MetaDescription: "Panduan praktis kaidah SEO website modern di tahun 2026. Optimasi Core Web Vitals, Structured Data, arsitektur teknis, dan strategi konten berkualitas tinggi.",
			MetaKeywords: "panduan seo 2026, optimasi website google, seo software house, cara ranking 1 google, core web vitals indonesia",
			CanonicalURL: "https://kotban.com/blog/panduan-lengkap-seo-website-2026",
			PublishedAt: &twoDaysAgo,
			Content: `## Mengapa Kaidah SEO 2026 Berbeda dari Tahun-Tahun Sebelumnya?

Perkembangan mesin pencari Google dan teknologi kecerdasan buatan (AI Search Experience) telah mengubah lanskap optimasi mesin pencari secara drastis. Saat ini, menerapkan teknik *keyword stuffing* atau *backlink spamming* bukan hanya tidak efektif, melainkan dapat membuat website Anda terkena penalti algoritma secara permanen.

Google kini menuntut **E-E-A-T** (*Experience, Expertise, Authoritativeness, and Trustworthiness*) yang nyata serta performa website berkecepatan tinggi yang lolos metrik **Core Web Vitals**.

---

## 1. Fondasi Teknis (Technical SEO) yang Wajib Diterapkan

Technical SEO adalah fondasi terpenting sebelum Anda menulis ribuan kata konten. Tanpa fondasi teknis yang kokoh, Googlebot akan kesulitan merayapi (*crawling*) dan mengindeks (*indexing*) halaman web Anda.

### A. Semantic HTML5 & Hierarki Heading
Gunakan elemen HTML5 yang jelas dan semantik:
- Gunakan tepat **satu tag \x60<h1>\x60** per halaman yang memuat *primary keyword*.
- Strukturkan subtopik dengan tag \x60<h2>\x60 dan \x60<h3>\x60 secara logis dan berurutan.
- Gunakan tag \x60<article>\x60, \x60<header>\x60, \x60<time>\x60, dan \x60<figure>\x60 agar mesin pencari mengenali bagian-bagian artikel berita Anda.

### B. Structured Data JSON-LD (Schema.org)
Implementasikan skema JSON-LD untuk membantu Google memahami konteks konten secara mendalam. Skema wajib untuk portal artikel meliputi:
- **BlogPosting / NewsArticle**: Menyediakan info judul, tanggal terbit, penulis, dan gambar cover untuk Google Discover dan Rich Snippets.
- **BreadcrumbList**: Menghasilkan navigasi breadcrumb langsung di hasil pencarian Google.
- **Organization**: Membangun otoritas entitas merek bisnis Anda.

![Diagram Arsitektur Structured Data & Validasi SEO Google](https://images.unsplash.com/photo-1504868584819-f8e8b4b6d7e3?w=1000&auto=format&fit=crop&q=80)

---

## 2. Optimasi Core Web Vitals & Kecepatan Rendering

Kecepatan bukan lagi sekadar faktor kenyamanan pengguna, melainkan faktor penentu peringkat resmi Google:
1. **LCP (Largest Contentful Paint)**: Harus di bawah 2.5 detik. Optimalkan format gambar ke WebP/AVIF modern dan gunakan CDN.
2. **INP (Interaction to Next Paint)**: Memastikan interaktivitas tombol dan navigasi instan di bawah 200 milidetik.
3. **CLS (Cumulative Layout Shift)**: Hindari pergeseran layout mendadak saat elemen web sedang dimuat.

![Analisis Skor Google Core Web Vitals & Performa Website Bisnis](https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=1000&auto=format&fit=crop&q=80)

---

## 3. Strategi Konten Berbasis Search Intent

Tulis konten yang menjawab secara langsung pertanyaan pengunjung. Buat struktur artikel yang mudah dipindai (*scannable*) menggunakan:
- Ringkasan di awal artikel (*quick takeaways*).
- Daftar Poin & Tabel perbandingan data.
- Contoh studi kasus nyata yang relevan dengan industri di Indonesia.

---

## Kesimpulan

Membangun website dengan kaidah SEO terbaik sejak tahap awal pengembangan adalah investasi jangka panjang paling menguntungkan bagi bisnis Anda. Di **Kotban.com**, seluruh website dan aplikasi yang kami kembangkan dirancang dengan arsitektur SEO-first berkinerja tinggi.

> **Ingin memiliki website bisnis yang cepat, elegan, dan langsung siap menduduki halaman 1 Google?** [Konsultasikan kebutuhan proyek Anda bersama tim ahli Kotban.com sekarang juga!](/konsultasi)`,
		},
		{
			Title:       "Mengapa Bisnis Berkembang Butuh Website Custom Dibandingkan Template Biasa?",
			Slug:        "mengapa-bisnis-butuh-website-custom-bukan-template",
			Excerpt:     "Ketahui perbedaan mendasar antara website custom code dengan website berbasis template instan, serta dampaknya terhadap skalabilitas, keamanan, dan reputasi merek bisnis Anda.",
			Category:    "Web Development",
			Tags:        "Web Development, Custom Website, Bisnis Digital, Software House, Next.js",
			AuthorName:  "Tim Engineering Kotban",
			AuthorAvatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=150&auto=format&fit=crop&q=80",
			AuthorRole:  "Software Architecture Team",
			CoverImage:  "https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=1200&auto=format&fit=crop&q=80",
			Status:      "published",
			IsFeatured:  true,
			IsTrending:  false,
			ViewsCount:  980,
			ReadingTime: 5,
			MetaTitle:   "Website Custom vs Template untuk Bisnis Berkembang | Kotban.com",
			MetaDescription: "Ketahui keunggulan website custom buatan software house profesional dibandingkan template instan untuk pertumbuhan bisnis jangka panjang.",
			MetaKeywords: "jasa pembuatan website custom, website custom vs template, software house indonesia, web developer profesional",
			CanonicalURL: "https://kotban.com/blog/mengapa-bisnis-butuh-website-custom-bukan-template",
			PublishedAt: &threeDaysAgo,
			Content: `## Fenomena Template Instan vs Kebutuhan Nyata Bisnis

Banyak pemilik bisnis tergoda menggunakan template website siap pakai karena biaya awal yang terlihat murah. Namun, seiring berkembangnya skala bisnis, keterbatasan teknis dan performa template sering kali menjadi hambatan besar.

Berikut adalah perbandingan mendalam mengapa perusahaan dan bisnis modern beralih ke solusi **Website Custom Code**:

---

## 1. Performa Super Cepat & Tanpa Beban Kode Berlebih (Bloatware)

Template instan biasanya dibuat untuk melayani ribuan skenario sekaligus. Akibatnya, website memuat ratusan script CSS dan JavaScript yang sebenarnya tidak digunakan oleh bisnis Anda.

Sebaliknya, **Website Custom**:
- Dibangun khusus sesuai alur kerja (*business logic*) dan fitur yang Anda perlukan.
- Waktu loading lebih cepat hingga 70% dibandingkan template WordPress generik.
- Skor Google PageSpeed Insights mencapai 90-100 secara konsisten.

---

## 2. Keamanan Tingkat Lanjut & Beban Pemeliharaan Rendah

Plugin pihak ketiga pada template CMS instan adalah celah keamanan terbesar di dunia internet saat ini. Ketika ada kerentanan (*vulnerability*) pada salah satu plugin, seluruh data pelanggan dan transaksi bisnis Anda bisa bocor.

Dengan arsitektur modern seperti **Next.js & Go Backend**:
- Kode terenkripsi dan berjalan di serverless/microservices yang aman.
- Tanpa risiko inject database atau plugin berbahaya.
- Menjamin integritas data transaksi dan privasi pelanggan.

---

## 3. Integrasi Fleksibel dengan Sistem Bisnis Internal

Bisnis yang sedang tumbuh membutuhkan integrasi ke berbagai sistem:
- Payment Gateway lokal (QRIS, Virtual Account, E-Wallet).
- Kurir & Logistik API (JNE, SiCepat, J&T).
- Sistem CRM, ERP, atau sistem pembukuan akuntansi internal.

Website custom memberikan kebebasan 100% untuk menghubungkan API apa pun tanpa batasan arsitektur.

---

## Siap Naik Kelas Bersama Kotban.com?

Tim **Kotban.com** siap merancang dan membangun website custom berstandar industri dengan teknologi modern paling mutakhir. Hubungi tim kami untuk sesi konsultasi gratis sekarang juga!`,
		},
		{
			Title:       "Next.js vs React: Mana Pilihan Terbaik untuk Proyek Website Bisnis Anda?",
			Slug:        "nextjs-vs-react-mana-terbaik-untuk-proyek-web",
			Excerpt:     "Ulasan mendalam mengenai perbedaan arsitektur Next.js Server Components vs Client-Side React SPA, performa SEO, dan rekomendasi stack untuk kebutuhan software perusahaan.",
			Category:    "Web Development",
			Tags:        "Next.js, React, Frontend, SSR, Performa Web",
			AuthorName:  "Taqwim",
			AuthorAvatar: "https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=150&auto=format&fit=crop&q=80",
			AuthorRole:  "Lead SEO & Fullstack Architect",
			CoverImage:  "https://images.unsplash.com/photo-1555066931-4365d14bab8c?w=1200&auto=format&fit=crop&q=80",
			Status:      "published",
			IsFeatured:  false,
			IsTrending:  true,
			ViewsCount:  1240,
			ReadingTime: 6,
			MetaTitle:   "Next.js vs React: Mana yang Tepat untuk Proyek Bisnis? | Kotban.com",
			MetaDescription: "Panduan memilih antara Next.js dan React untuk website bisnis Anda. Perbandingan SSR vs CSR, kapabilitas SEO, dan kecepatan muat halaman.",
			MetaKeywords: "next.js vs react, perbedaan nextjs dan react, jasa nextjs indonesia, frontend developer software house",
			CanonicalURL: "https://kotban.com/blog/nextjs-vs-react-mana-terbaik-untuk-proyek-web",
			PublishedAt: &fiveDaysAgo,
			Content: `## Pengantar: Ekosistem Modern Frontend Web

Dalam dunia pengembangan web modern, React dan Next.js adalah dua nama yang paling sering diperbincangkan. Namun, masih banyak pengambil keputusan bisnis yang bingung: *kapan harus menggunakan React murni, dan kapan wajib menggunakan Next.js?*

Mari kita bedah secara komprehensif.

---

## 1. Perbedaan Mendasar: Library vs Full-Stack Framework

- **React (Vite / CRA)**: Adalah *library UI* berbasis Client-Side Rendering (CSR). Browser mengunduh file JavaScript kosong terlebih dahulu, lalu merender konten di sisi pengguna.
- **Next.js**: Adalah *framework lengkap* berbasis React yang mendukung Server-Side Rendering (SSR), Static Site Generation (SSG), dan Server Components secara bawaan.

---

## 2. Faktor Penentu: Kemampuan SEO & Kecepatan Indexing

| Aspek | React (SPA) | Next.js (SSR / SSG) |
| :--- | :--- | :--- |
| **SEO Indexing** | Tergantung kemampuan bot mengeksekusi JS | **Instan & Sempurna**, HTML langsung tersedia |
| **First Contentful Paint** | Lebih lambat pada koneksi lemah | **Sangat Cepat**, konten sudah ter-render di server |
| **Social Sharing (OG Tag)** | Sulit dinamis tanpa prerender | **Dukungan Penuh Dynamic Metadata** |
| **Cocok Untuk** | Dashboard internal, SaaS portal tertutup | **Landing Page, Portal Berita, Toko Online** |

---

## 3. Kapan Anda Harus Memilih Next.js?

Pilihlah Next.js jika proyek Anda membutuhkan:
1. Peringkat SEO tinggi di Google untuk mendatangkan traffic organik pelanggan.
2. Kecepatan muat halaman instan di semua jenis perangkat mobile.
3. Halaman publik seperti Company Profile, Portal Berita, Katalog E-Commerce, dan Landing Page penawaran jasa.

Di **Kotban.com**, kami menggunakan Next.js 16 App Router dengan React 19 untuk menghasilkan performa maksimal bagi klien kami.`,
		},
		{
			Title:       "Tren Pengembangan Aplikasi Mobile 2026: Peluang Emas Transformasi Digital",
			Slug:        "tren-aplikasi-mobile-2026-dan-peluang-bisnis",
			Excerpt:     "Ketahui tren aplikasi mobile terkini tahun 2026 mulai dari Cross-Platform Flutter/React Native, integrasi AI On-Device, hingga arsitektur micro-apps untuk efisiensi operasional.",
			Category:    "Mobile App",
			Tags:        "Mobile App, Flutter, React Native, Android, iOS, Transformasi Digital",
			AuthorName:  "Tim Engineering Kotban",
			AuthorAvatar: "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=150&auto=format&fit=crop&q=80",
			AuthorRole:  "Mobile Development Lead",
			CoverImage:  "https://images.unsplash.com/photo-1512941937669-90a1b58e7e9c?w=1200&auto=format&fit=crop&q=80",
			Status:      "published",
			IsFeatured:  false,
			IsTrending:  true,
			ViewsCount:  870,
			ReadingTime: 5,
			MetaTitle:   "Tren Pengembangan Aplikasi Mobile 2026 untuk Bisnis | Kotban.com",
			MetaDescription: "Ketahui tren pengembangan aplikasi mobile Android & iOS 2026 untuk memperluas jangkauan pasar dan meningkatkan retensi pelanggan bisnis Anda.",
			MetaKeywords: "jasa pembuatan aplikasi mobile, aplikasi android ios bisnis, developer flutter indonesia, software house mobile",
			CanonicalURL: "https://kotban.com/blog/tren-aplikasi-mobile-2026-dan-peluang-bisnis",
			PublishedAt: &oneWeekAgo,
			Content: `## Mengapa Aplikasi Mobile Semakin Krusial di 2026?

Lebih dari 85% waktu pengguna internet di Indonesia dihabiskan di dalam aplikasi smartphone. Bisnis yang hanya mengandalkan saluran konvensional semakin tertinggal dari kompetitor yang memiliki aplikasi mobile mandiri.

Berikut tren utama aplikasi mobile 2026 yang perlu Anda manfaatkan:

---

## 1. Efisiensi Biaya dengan Cross-Platform (Flutter & React Native)

Mengembangkan dua basis kode terpisah (Swift untuk iOS dan Kotlin untuk Android) sering kali memakan biaya dan waktu dua kali lipat.
Dengan framework cross-platform modern:
- Satu basis kode berjalan mulus di Android dan iOS secara bersamaan.
- Waktu rilis ke pasar (*Time to Market*) lebih cepat 50%.
- Biaya pemeliharaan (*maintenance*) jauh lebih hemat dan efisien.

---

## 2. Personalisasi & AI On-Device

Aplikasi mobile kini mampu memberikan rekomendasi produk dan notifikasi cerdas berbasis kebiasaan pengguna secara *real-time*, sehingga meningkatkan rasio konversi hingga 300%.

---

## Bangun Aplikasi Mobile Bisnis Anda Bersama Kami

Kotban.com berpengalaman membangun aplikasi mobile kelas enterprise untuk berbagai industri: retail POS, reservasi klinik, logistik armada, hingga platform komunitas edukasi. Hubungi kami untuk konsultasi teknis!`,
		},
		{
			Title:       "Manfaat Nyata Sistem Informasi & ERP Kustom untuk Meningkatkan Efisiensi Bisnis",
			Slug:        "manfaat-sistem-informasi-erp-untuk-umkm",
			Excerpt:     "Pelajari bagaimana implementasi sistem informasi manajemen dan ERP custom dapat memangkas biaya operasional, mencegah kebocoran inventori, dan mempercepat pengambilan keputusan.",
			Category:    "Sistem Informasi",
			Tags:        "Sistem Informasi, ERP Custom, Manajemen Bisnis, Otomasi Bisnis",
			AuthorName:  "Tim Redaksi Kotban",
			AuthorAvatar: "https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=150&auto=format&fit=crop&q=80",
			AuthorRole:  "Business Systems Analyst",
			CoverImage:  "https://images.unsplash.com/photo-1454165804606-c3d57bc86b40?w=1200&auto=format&fit=crop&q=80",
			Status:      "published",
			IsFeatured:  false,
			IsTrending:  false,
			ViewsCount:  650,
			ReadingTime: 6,
			MetaTitle:   "Manfaat Sistem Informasi & ERP Kustom untuk Efisiensi Bisnis | Kotban.com",
			MetaDescription: "Tingkatkan efisiensi dan kontrol operasional bisnis Anda dengan sistem informasi dan ERP custom buatan Kotban.com.",
			MetaKeywords: "jasa pembuatan sistem informasi, software erp custom, aplikasi manajemen gudang, software house sistem informasi",
			CanonicalURL: "https://kotban.com/blog/manfaat-sistem-informasi-erp-untuk-umkm",
			PublishedAt: &oneWeekAgo,
			Content: `## Tantangan Operasional Bisnis yang Berkembang

Ketika transaksi harian bisnis Anda bertambah dari puluhan menjadi ratusan atau ribuan, pencatatan manual via spreadsheet (Excel) mulai memunculkan masalah fatal:
- Selisih stok gudang yang sulit dilacak.
- Laporan laba-rugi yang terlambat berminggu-minggu.
- Duplikasi pekerjaan antar departemen (sales, gudang, keuangan).

---

## Keuntungan Menggunakan Sistem ERP Kustom

1. **Visibilitas Real-Time Seluruh Cabang**: Pantau arus kas, performa penjualan, dan posisi stok langsung dari dashboard satu pintu.
2. **Otomatisasi Laporan Keuangan**: Laporan neraca, laba-rugi, dan arus kas terhitung otomatis tanpa risiko kesalahan manusia (*human error*).
3. **Akses Berjenjang Berbasis Peran (Role-Based Access)**: Lindungi data rahasia perusahaan dengan membatasi akses sesuai jabatan staf Anda.

---

## Solusi Software Terintegrasi dari Kotban.com

Kami merancang sistem ERP dan Software Manajemen yang menyesuaikan SOP bisnis Anda, bukan memaksa bisnis Anda menyesuaikan software yang kaku. [Jadwalkan sesi demo sistem sekarang](/konsultasi).`,
		},
	}

	for _, a := range articles {
		db.Create(&a)
	}
	log.Println("📰 Seeded default SEO articles for Blog & Portal Berita")
}

