package database

import (
	"log"
	"time"

	"github.com/tstech/backend/internal/config"
	"github.com/tstech/backend/internal/model"
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
		&model.Category{},
		&model.Service{},
		&model.Product{},
		&model.SaaSProduct{},
		&model.SaaSPlan{},
		&model.SaaSSubscription{},
		&model.Invoice{},
		&model.AuditLog{},
		&model.AIBlogSetting{},
	)
	if err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("✅ Database migrations completed")

	// Ensure is_featured is true for SaaS products if null
	db.Exec("UPDATE saas_products SET is_featured = true WHERE is_featured IS NULL")

	// Auto seed SaaS Products & Plans
	seedSaaSProducts(db)
	seedSampleSubscription(db)
	syncDefaultPlanModules(db)

	// Auto seed Products
	seedProducts(db)

	// Auto seed Services
	seedServices(db)

	// Auto seed Categories
	seedCategories(db)

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

	// Auto seed Gemini AI Blog Setting
	seedAIBlogSetting(db)
}

func seedAIBlogSetting(db *gorm.DB) {
	var count int64
	db.Model(&model.AIBlogSetting{}).Count(&count)
	if count == 0 {
		setting := model.AIBlogSetting{
			IsEnabled:      false,
			GeminiModel:    "gemini-3-flash-preview",
			IntervalHours:  24,
			TargetCategory: "Teknologi",
			DefaultStatus:  "published",
			AutoSEO:        true,
			LastRunStatus:  "idle",
			TotalGenerated: 0,
		}
		db.Create(&setting)
		log.Println("🤖 Seeded default Gemini AI Blog settings (gemini-3-flash-preview)")
	} else {
		// Auto upgrade any legacy gemini-1.5, 2.0, or 2.5 models in database to gemini-3-flash-preview
		db.Model(&model.AIBlogSetting{}).
			Where("gemini_model LIKE ? OR gemini_model LIKE ? OR gemini_model LIKE ? OR gemini_model = '' OR gemini_model IS NULL", "%1.5%", "%2.0%", "%2.5%").
			Update("gemini_model", "gemini-3-flash-preview")
	}
}

func seedAdmin(db *gorm.DB, cfg *config.Config) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(cfg.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("⚠️ Failed to hash admin password: %v", err)
		return
	}

	adminEmails := []string{cfg.AdminEmail, "admin@tstech.id", "admin@tstech.com"}
	for _, email := range adminEmails {
		if email == "" {
			continue
		}
		var user model.User
		if err := db.Where("email = ?", email).First(&user).Error; err != nil {
			newAdmin := model.User{
				Email:       email,
				Password:    string(hashed),
				Name:        cfg.AdminName,
				Role:        "admin",
				Phone:       cfg.WhatsAppNumber,
				CompanyName: "TSTech",
				IsActive:    true,
			}
			if err := db.Create(&newAdmin).Error; err != nil {
				log.Printf("⚠️ Failed to seed admin user (%s): %v", email, err)
			} else {
				log.Printf("👑 Seeded admin user: %s (Password: %s)", email, cfg.AdminPassword)
			}
		} else {
			user.Password = string(hashed)
			user.Role = "admin"
			user.IsActive = true
			db.Save(&user)
			log.Printf("👑 Refreshed admin credentials for: %s (Password: %s)", email, cfg.AdminPassword)
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
			Value: "TsTech adalah software house modern yang berfokus pada pengembangan solusi digital berkualitas tinggi, scalable, dan tepat waktu untuk berbagai kebutuhan bisnis.",
			Type:  "text",
			Group: "about",
		},
		{
			Key:   "contact_email",
			Value: "halo@tstech.id",
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
			Value: `[{"category":"Umum","items":[{"question":"Apa itu TsTech?","answer":"TsTech adalah software house profesional yang melayani jasa pembuatan website, aplikasi mobile (Android/iOS), sistem informasi custom (ERP/CRM/POS), UI/UX design, serta pemeliharaan software untuk berbagai skala bisnis."},{"question":"Layanan apa saja yang tersedia?","answer":"Kami menyediakan pembuatan Website Company Profile, Toko Online (E-Commerce), Aplikasi Mobile (Android/iOS), Sistem Informasi Custom (ERP, CRM, HRIS), Maintenance & Support, serta Jasa Desain UI/UX."}]},{"category":"Proses & Waktu Pengerjaan","items":[{"question":"Berapa lama waktu pengerjaan website/aplikasi?","answer":"Estimasi waktu pengerjaan bervariasi tergantung skala proyek. Paket Basic membutuhkan 7-14 hari kerja, Professional 14-30 hari kerja, dan Enterprise 30-60 hari kerja."},{"question":"Bagaimana proses revisi?","answer":"Revisi dilakukan pada tahap testing & QA. Jumlah revisi menyesuaikan paket yang dipilih (Basic: 1x, Professional: 3x, Enterprise: Unlimited sesuai ruang lingkup brief awal)."}]},{"category":"Pembayaran","items":[{"question":"Berapa besar DP yang dibutuhkan?","answer":"DP (Down Payment) standar adalah sebesar 50% dari total nilai proyek saat pesanan dibuat, dan pelunasan 50% sisanya dilakukan setelah testing selesai dan produk siap dilaunching."},{"question":"Metode pembayaran apa saja yang diterima?","answer":"Kami menerima pembayaran via iPaymu Payment Gateway yang mendukung Bank Transfer (Virtual Account), QRIS, E-Wallet, serta Kartu Kredit."}]},{"category":"Garansi & Hak Milik","items":[{"question":"Apakah klien mendapatkan source code dan hak milik penuh?","answer":"Ya, 100% hak cipta, source code, dan akses server akan diserahkan sepenuhnya kepada klien setelah pelunasan."},{"question":"Apakah ada garansi setelah selesai?","answer":"Kami memberikan garansi perbaikan bug dan error gratis selama 30 hingga 180 hari sesuai paket yang dipilih."}]}]`,
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
				Slug:               "e-commerce-fashion-brand",
				Title:              "Platform E-Commerce Fashion Brand",
				Category:           "Website",
				ClientName:         "Zola Official",
				Industry:           "Retail & Fashion",
				Problem:            "Klien membutuhkan toko online dengan handling order ribuan transaksi per hari.",
				Solution:           "Kami membangun web store berbasis Next.js & Go microservice terintegrasi payment gateway.",
				TechStack:          "Next.js, Go, PostgreSQL, Redis, iPaymu",
				Result:             "Peningkatan transaksi 180% dan loading time di bawah 1 detik.",
				DemoURL:            "https://example.com",
				IsFeatured:         true,
				SortOrder:          1,
				TestimonialQuote:   "I am very satisfied with the result, it is exactly what I wanted, you did a very good job!",
				TestimonialAuthor:  "Diaby Mamadou",
				TestimonialRole:    "Founder & CTO",
				TestimonialCompany: "Genesys - Paris, France",
			},
			{
				Slug:               "pos-sistem-kasir-restoran",
				Title:              "Aplikasi Mobile POS & Manajemen Restoran",
				Category:           "Mobile App",
				ClientName:         "Kopi Nusantara Group",
				Industry:           "F&B",
				Problem:            "Pencatatan kasir manual antar 15 cabang rawan selisih stok.",
				Solution:           "Aplikasi tablet & mobile kasir tersinkronisasi realtime ke cloud dashboard.",
				TechStack:          "Flutter, Go, PostgreSQL, WebSocket",
				Result:             "Efisiensi order 40% lebih cepat dan laporan keuangan realtime.",
				DemoURL:            "https://example.com",
				IsFeatured:         true,
				SortOrder:          2,
				TestimonialQuote:   "Sistem POS dan sinkronisasi cloud yang dibangun TsTech sangat stabil saat jam sibuk di seluruh cabang kami.",
				TestimonialAuthor:  "Hendro Kusumo",
				TestimonialRole:    "Operational Director",
				TestimonialCompany: "Kopi Nusantara - Jakarta",
			},
			{
				Slug:               "sistem-erp-manufaktur",
				Title:              "Sistem ERP Custom Pabrik & Gudang",
				Category:           "Sistem Informasi",
				ClientName:         "PT Surya Logistik",
				Industry:           "Manufaktur & Logistik",
				Problem:            "Manajemen inventori ribuan SKU yang tersebar di 4 gudang.",
				Solution:           "Sistem ERP berbasis web terintegrasi barcode scanner & auto-purchase order.",
				TechStack:          "React, Echo Go, PostgreSQL, Docker",
				Result:             "Reduksi human error hingga 95% dan audit stock opname realtime.",
				DemoURL:            "https://example.com",
				IsFeatured:         true,
				SortOrder:          3,
				TestimonialQuote:   "The ERP platform reduced our warehouse tracking error by 95%. Incredible engineering quality and responsiveness.",
				TestimonialAuthor:  "Sarah Jenkins",
				TestimonialRole:    "Head of Supply Chain",
				TestimonialCompany: "Logistics Hub International",
			},
		}
		for _, s := range samples {
			db.Create(&s)
		}
		log.Println("🎨 Seeded sample portfolios with client testimonials")
	}

	var tCount int64
	db.Model(&model.Testimonial{}).Count(&tCount)
	if tCount == 0 {
		testimoniList := []model.Testimonial{
			{
				ClientName:  "Ahmad Ridwan",
				CompanyName: "PT Maju Bersama",
				Rating:      5,
				Content:     "TsTech membantu kami membuat website e-commerce yang sangat profesional. Penjualan online kami meningkat 150% setelah launch!",
				IsActive:    true,
				SortOrder:   1,
			},
			{
				ClientName:  "Sari Dewi",
				CompanyName: "Klinik Sehat",
				Rating:      5,
				Content:     "Sistem informasi klinik yang dibuat sangat memudahkan operasional kami. Tim TsTech sangat responsif dan profesional.",
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
		{Key: "manual_bank_accounts", Value: `[{"bank_name":"BCA","account_number":"8735098231","account_holder":"PT TSTECH SOLUSI TEKNOLOGI","icon":"bca"},{"bank_name":"Bank Mandiri","account_number":"1370019827364","account_holder":"PT TSTECH SOLUSI TEKNOLOGI","icon":"mandiri"},{"bank_name":"Bank Syariah Indonesia (BSI)","account_number":"7219082341","account_holder":"PT TSTECH SOLUSI TEKNOLOGI","icon":"bsi"}]`},
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
			Title:           "Panduan Lengkap SEO Website 2026: Strategi Ampuh Raih Peringkat #1 Google",
			Slug:            "panduan-lengkap-seo-website-2026",
			Excerpt:         "Pelajari kaidah SEO terbaru 2026 mulai dari Core Web Vitals, Search Intent, Structured Data JSON-LD, hingga optimasi AI Search untuk mendominasi peringkat pertama Google.",
			Category:        "SEO & Web Optimization",
			Tags:            "SEO, Google Search, Core Web Vitals, Web Optimization, Search Intent",
			AuthorName:      "Taqwim",
			AuthorAvatar:    "https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=150&auto=format&fit=crop&q=80",
			AuthorRole:      "Lead SEO & Fullstack Architect",
			CoverImage:      "https://images.unsplash.com/photo-1460925895917-afdab827c52f?w=1200&auto=format&fit=crop&q=80",
			Status:          "published",
			IsFeatured:      true,
			IsTrending:      true,
			ViewsCount:      1420,
			ReadingTime:     7,
			MetaTitle:       "Panduan Lengkap SEO Website 2026: Strategi Ampuh Peringkat #1 | TsTech",
			MetaDescription: "Panduan praktis kaidah SEO website modern di tahun 2026. Optimasi Core Web Vitals, Structured Data, arsitektur teknis, dan strategi konten berkualitas tinggi.",
			MetaKeywords:    "panduan seo 2026, optimasi website google, seo software house, cara ranking 1 google, core web vitals indonesia",
			CanonicalURL:    "https://tstech.id/blog/panduan-lengkap-seo-website-2026",
			PublishedAt:     &twoDaysAgo,
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

Membangun website dengan kaidah SEO terbaik sejak tahap awal pengembangan adalah investasi jangka panjang paling menguntungkan bagi bisnis Anda. Di **TsTech**, seluruh website dan aplikasi yang kami kembangkan dirancang dengan arsitektur SEO-first berkinerja tinggi.

> **Ingin memiliki website bisnis yang cepat, elegan, dan langsung siap menduduki halaman 1 Google?** [Konsultasikan kebutuhan proyek Anda bersama tim ahli TsTech sekarang juga!](/konsultasi)`,
		},
		{
			Title:           "Mengapa Bisnis Berkembang Butuh Website Custom Dibandingkan Template Biasa?",
			Slug:            "mengapa-bisnis-butuh-website-custom-bukan-template",
			Excerpt:         "Ketahui perbedaan mendasar antara website custom code dengan website berbasis template instan, serta dampaknya terhadap skalabilitas, keamanan, dan reputasi merek bisnis Anda.",
			Category:        "Web Development",
			Tags:            "Web Development, Custom Website, Bisnis Digital, Software House, Next.js",
			AuthorName:      "Tim Engineering TsTech",
			AuthorAvatar:    "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=150&auto=format&fit=crop&q=80",
			AuthorRole:      "Software Architecture Team",
			CoverImage:      "https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=1200&auto=format&fit=crop&q=80",
			Status:          "published",
			IsFeatured:      true,
			IsTrending:      false,
			ViewsCount:      980,
			ReadingTime:     5,
			MetaTitle:       "Website Custom vs Template untuk Bisnis Berkembang | TsTech",
			MetaDescription: "Ketahui keunggulan website custom buatan software house profesional dibandingkan template instan untuk pertumbuhan bisnis jangka panjang.",
			MetaKeywords:    "jasa pembuatan website custom, website custom vs template, software house indonesia, web developer profesional",
			CanonicalURL:    "https://tstech.id/blog/mengapa-bisnis-butuh-website-custom-bukan-template",
			PublishedAt:     &threeDaysAgo,
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

## Perbandingan Langsung: Website Custom vs Template Instan

| Parameter Kunci | Website Custom (TsTech) | Website Template Instan |
| :--- | :--- | :--- |
| **Kecepatan & Performa** | Skor PageSpeed 95-100 (Clean Code) | Skor 40-70 (Banyak Bloatware) |
| **Keamanan Sistem** | Sangat Aman & Terenkripsi | Rentan Vulnerability Plugin |
| **Skalabilitas Fitur** | Bebas Integrasi API Apa Pun | Dibatasi Fitur Bawaan Template |
| **Kepemilikan Kode** | 100% Hak Milik Klien | Lisensi Sewa / Bergantung Tema |
| **Dampak SEO Google** | Terstruktur & Cepat Masuk Halaman #1 | Sering Lambat & Sulit Bersaing |

---

## Siap Naik Kelas Bersama TsTech?

Tim **TsTech** siap merancang dan membangun website custom berstandar industri dengan teknologi modern paling mutakhir. Hubungi tim kami untuk sesi konsultasi gratis sekarang juga!`,
		},
		{
			Title:           "Next.js vs React: Mana Pilihan Terbaik untuk Proyek Website Bisnis Anda?",
			Slug:            "nextjs-vs-react-mana-terbaik-untuk-proyek-web",
			Excerpt:         "Ulasan mendalam mengenai perbedaan arsitektur Next.js Server Components vs Client-Side React SPA, performa SEO, dan rekomendasi stack untuk kebutuhan software perusahaan.",
			Category:        "Web Development",
			Tags:            "Next.js, React, Frontend, SSR, Performa Web",
			AuthorName:      "Taqwim",
			AuthorAvatar:    "https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=150&auto=format&fit=crop&q=80",
			AuthorRole:      "Lead SEO & Fullstack Architect",
			CoverImage:      "https://images.unsplash.com/photo-1555066931-4365d14bab8c?w=1200&auto=format&fit=crop&q=80",
			Status:          "published",
			IsFeatured:      false,
			IsTrending:      true,
			ViewsCount:      1240,
			ReadingTime:     6,
			MetaTitle:       "Next.js vs React: Mana yang Tepat untuk Proyek Bisnis? | TsTech",
			MetaDescription: "Panduan memilih antara Next.js dan React untuk website bisnis Anda. Perbandingan SSR vs CSR, kapabilitas SEO, dan kecepatan muat halaman.",
			MetaKeywords:    "next.js vs react, perbedaan nextjs dan react, jasa nextjs indonesia, frontend developer software house",
			CanonicalURL:    "https://tstech.id/blog/nextjs-vs-react-mana-terbaik-untuk-proyek-web",
			PublishedAt:     &fiveDaysAgo,
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

Di **TsTech**, kami menggunakan Next.js 16 App Router dengan React 19 untuk menghasilkan performa maksimal bagi klien kami.`,
		},
		{
			Title:           "Tren Pengembangan Aplikasi Mobile 2026: Peluang Emas Transformasi Digital",
			Slug:            "tren-aplikasi-mobile-2026-dan-peluang-bisnis",
			Excerpt:         "Ketahui tren aplikasi mobile terkini tahun 2026 mulai dari Cross-Platform Flutter/React Native, integrasi AI On-Device, hingga arsitektur micro-apps untuk efisiensi operasional.",
			Category:        "Mobile App",
			Tags:            "Mobile App, Flutter, React Native, Android, iOS, Transformasi Digital",
			AuthorName:      "Tim Engineering TsTech",
			AuthorAvatar:    "https://images.unsplash.com/photo-1507003211169-0a1dd7228f2d?w=150&auto=format&fit=crop&q=80",
			AuthorRole:      "Mobile Development Lead",
			CoverImage:      "https://images.unsplash.com/photo-1512941937669-90a1b58e7e9c?w=1200&auto=format&fit=crop&q=80",
			Status:          "published",
			IsFeatured:      false,
			IsTrending:      true,
			ViewsCount:      870,
			ReadingTime:     5,
			MetaTitle:       "Tren Pengembangan Aplikasi Mobile 2026 untuk Bisnis | TsTech",
			MetaDescription: "Ketahui tren pengembangan aplikasi mobile Android & iOS 2026 untuk memperluas jangkauan pasar dan meningkatkan retensi pelanggan bisnis Anda.",
			MetaKeywords:    "jasa pembuatan aplikasi mobile, aplikasi android ios bisnis, developer flutter indonesia, software house mobile",
			CanonicalURL:    "https://tstech.id/blog/tren-aplikasi-mobile-2026-dan-peluang-bisnis",
			PublishedAt:     &oneWeekAgo,
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

TsTech berpengalaman membangun aplikasi mobile kelas enterprise untuk berbagai industri: retail POS, reservasi klinik, logistik armada, hingga platform komunitas edukasi. Hubungi kami untuk konsultasi teknis!`,
		},
		{
			Title:           "Manfaat Nyata Sistem Informasi & ERP Kustom untuk Meningkatkan Efisiensi Bisnis",
			Slug:            "manfaat-sistem-informasi-erp-untuk-umkm",
			Excerpt:         "Pelajari bagaimana implementasi sistem informasi manajemen dan ERP custom dapat memangkas biaya operasional, mencegah kebocoran inventori, dan mempercepat pengambilan keputusan.",
			Category:        "Sistem Informasi",
			Tags:            "Sistem Informasi, ERP Custom, Manajemen Bisnis, Otomasi Bisnis",
			AuthorName:      "Tim Redaksi TsTech",
			AuthorAvatar:    "https://images.unsplash.com/photo-1534528741775-53994a69daeb?w=150&auto=format&fit=crop&q=80",
			AuthorRole:      "Business Systems Analyst",
			CoverImage:      "https://images.unsplash.com/photo-1454165804606-c3d57bc86b40?w=1200&auto=format&fit=crop&q=80",
			Status:          "published",
			IsFeatured:      false,
			IsTrending:      false,
			ViewsCount:      650,
			ReadingTime:     6,
			MetaTitle:       "Manfaat Sistem Informasi & ERP Kustom untuk Efisiensi Bisnis | TsTech",
			MetaDescription: "Tingkatkan efisiensi dan kontrol operasional bisnis Anda dengan sistem informasi dan ERP custom buatan TsTech.",
			MetaKeywords:    "jasa pembuatan sistem informasi, software erp custom, aplikasi manajemen gudang, software house sistem informasi",
			CanonicalURL:    "https://tstech.id/blog/manfaat-sistem-informasi-erp-untuk-umkm",
			PublishedAt:     &oneWeekAgo,
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

## Solusi Software Terintegrasi dari TsTech

Kami merancang sistem ERP dan Software Manajemen yang menyesuaikan SOP bisnis Anda, bukan memaksa bisnis Anda menyesuaikan software yang kaku. [Jadwalkan sesi demo sistem sekarang](/konsultasi).`,
		},
	}

	for _, a := range articles {
		db.Create(&a)
	}
	log.Println("📰 Seeded default SEO articles for Blog & Portal Berita")
}

func seedCategories(db *gorm.DB) {
	var count int64
	db.Model(&model.Category{}).Count(&count)
	if count > 0 {
		return
	}

	categories := []model.Category{
		// Blog Categories
		{Type: "blog", Name: "Web Development", Slug: "web-development", Description: "Wawasan seputar pengembangan website, web app modern, dan framework.", Icon: "🌐", SortOrder: 1, IsActive: true},
		{Type: "blog", Name: "Mobile App", Slug: "mobile-app", Description: "Pengembangan aplikasi Android dan iOS native maupun cross-platform.", Icon: "📱", SortOrder: 2, IsActive: true},
		{Type: "blog", Name: "SEO & Web Optimization", Slug: "seo-web-optimization", Description: "Strategi optimasi mesin pencari, Core Web Vitals, dan Search Intent.", Icon: "🚀", SortOrder: 3, IsActive: true},
		{Type: "blog", Name: "Sistem Informasi", Slug: "sistem-informasi", Description: "Implementasi ERP, CRM, dan software manajemen operasional bisnis.", Icon: "⚙️", SortOrder: 4, IsActive: true},
		{Type: "blog", Name: "Tips Bisnis", Slug: "tips-bisnis", Description: "Panduan strategi digitalisasi, efisiensi operasional, dan ROI teknologi.", Icon: "💡", SortOrder: 5, IsActive: true},
		{Type: "blog", Name: "UI/UX Design", Slug: "ui-ux-design", Description: "Prinsip desain antarmuka, user experience, prototyping, dan design system.", Icon: "🎨", SortOrder: 6, IsActive: true},
		{Type: "blog", Name: "Teknologi", Slug: "teknologi", Description: "Kabar tren arsitektur cloud, microservices, AI, dan keamanan siber.", Icon: "⚡", SortOrder: 7, IsActive: true},

		// Portfolio Categories
		{Type: "portfolio", Name: "Website & Web App", Slug: "website", Description: "Website company profile, web app SaaS, dan landing page konversi tinggi.", Icon: "🌐", SortOrder: 1, IsActive: true},
		{Type: "portfolio", Name: "Mobile Application", Slug: "mobile-app", Description: "Aplikasi mobile iOS dan Android berkinerja tinggi untuk pengguna aktif.", Icon: "📱", SortOrder: 2, IsActive: true},
		{Type: "portfolio", Name: "E-Commerce Platform", Slug: "e-commerce", Description: "Toko online terintegrasi payment gateway dan kurir logistik otomatis.", Icon: "🛒", SortOrder: 3, IsActive: true},
		{Type: "portfolio", Name: "Sistem Informasi & ERP", Slug: "sistem-informasi", Description: "Software custom otomasi bisnis, inventori pergudangan, dan keuangan.", Icon: "⚙️", SortOrder: 4, IsActive: true},
		{Type: "portfolio", Name: "UI/UX & Design System", Slug: "ui-ux", Description: "Desain prototype interaktif, visual branding, dan UI mockup figma.", Icon: "🎨", SortOrder: 5, IsActive: true},
	}

	for _, c := range categories {
		db.Create(&c)
	}
	log.Println("🏷️ Seeded default categories for Blog & Portfolio")
}

func seedServices(db *gorm.DB) {
	var count int64
	db.Model(&model.Service{}).Count(&count)
	if count > 0 {
		return
	}

	services := []model.Service{
		{
			Slug:            "web-application-development-services",
			Title:           "Web Application Development Services",
			Tagline:         "Kami mengembangkan aplikasi web handal pada berbagai platform teknologi open-source mutakhir untuk memastikan skalabilitas tinggi dan pengelolaan anggaran yang efisien.",
			Badge:           "Web Application Development",
			Icon:            "🌐",
			OverviewTitle:   "Scalable and Custom Web Application Development",
			OverviewContent: "TsTech memanfaatkan teknologi web modern dan praktik engineering terbaik untuk membangun aplikasi web berkinerja tinggi, aman, dan dapat diskalakan yang memberikan hasil bisnis nyata. Jika Anda menginginkan aplikasi web yang user-friendly, intuitif, cepat, dan elegan di saat bersamaan – Anda berada di tempat yang tepat.\n\nKami mengutamakan efisiensi dalam setiap custom web application dengan menerapkan standar desain industri, clean architecture, dan pengujian ketat (QA & automated testing). Dengan begitu, klien kami menerima aplikasi web andal yang siap digunakan sejak hari pertama peluncuran. Penerapan metodologi agile memastikan proyek diselesaikan tepat waktu sesuai ruang lingkup dan anggaran.\n\nAplikasi web kustom menjawab keterbatasan software instan/template dengan memberikan fleksibilitas alur kerja (workflow), keamanan data tingkat enterprise, integrasi multi-sistem (API & database), dan performa tanpa beban bloatware.",
			OverviewImage:   "https://images.unsplash.com/photo-1460925895917-afdab827c52f?w=1000&auto=format&fit=crop&q=80",
			ProcessTitle:    "Development Process",
			ProcessContent:  "• **1. Planning & Analisis Kebutuhan**: Tahap fundamental untuk memetakan tujuan bisnis, alur kerja sistem, spesifikasi fitur, dan ekspektasi performa.\n• **2. UI/UX Design & Prototyping**: Perancangan antarmuka visual interaktif di Figma, pengujian alur pengguna (*user journey*), dan review bersama klien sebelum tahap coding.\n• **3. Web Development & Architecture**: Implementasi frontend dan backend menggunakan clean architecture, API modular, dan standar keamanan data modern.\n• **4. Comprehensive Testing & QA**: Pengujian fungsionalitas menyeluruh di berbagai browser dan resolusi layar, load testing, serta penanganan bug/vulnerabilitas.\n• **5. Deployment & Handover**: Peluncuran aplikasi ke server production (Cloud/VPS), konfigurasi domain & SSL, serah terima source code, dan pelatihan operasional.",
			ProcessImage:    "https://images.unsplash.com/photo-1531403009284-440f080d1e12?w=800&auto=format&fit=crop&q=80",
			TechTitle:       "Technologies & Tech Stack",
			TechContent:     "### Frontend\n- Next.js (App Router, React 19)\n- Vue.js / Nuxt.js\n- React.js / Vite\n- TypeScript & TailwindCSS\n\n### Backend\n- Go (Echo / Gin / Fiber)\n- Node.js (NestJS / Express)\n- PHP (Laravel Framework)\n- Python (FastAPI / Django)\n\n### Database & Cache\n- PostgreSQL & MySQL\n- MongoDB & Firebase\n- Redis (Caching & Rate Limiting)\n\n### Cloud & DevOps\n- Docker & Containerization\n- Linux VPS / AWS / Cloudflare / Vercel\n- CI/CD Automated Deployment",
			TechImage:       "https://images.unsplash.com/photo-1555066931-4365d14bab8c?w=800&auto=format&fit=crop&q=80",
			CtaTitle:        "Siap Mewujudkan Aplikasi Web Impian Bisnis Anda?",
			CtaDescription:  "Konsultasikan ide dan kebutuhan sistem Anda bersama tim lead architect TsTech. Dapatkan estimasi biaya transparan dan timeline pengerjaan yang jelas.",
			CtaButtonText:   "Konsultasi Gratis Sekarang",
			CtaButtonURL:    "/konsultasi",
			SortOrder:       1,
			IsActive:        true,
			MetaTitle:       "Jasa Pembuatan Web Application Custom & Profesional | TsTech",
			MetaDescription: "Jasa pengembangan web application scalable, aman, dan modern berbasis Next.js, Go, React, dan Laravel dengan arsitektur enterprise oleh TsTech.",
			MetaKeywords:    "web application development, jasa pembuatan web app, custom web app indonesia, software house web application",
		},
		{
			Slug:            "mobile-app-development",
			Title:           "Mobile Application Development (Android & iOS)",
			Tagline:         "Bangun aplikasi mobile native atau cross-platform berkinerja tinggi, responsif, dan interaktif untuk memperluas jangkauan pengguna dan meningkatkan loyalitas pelanggan.",
			Badge:           "Mobile App Development",
			Icon:            "📱",
			OverviewTitle:   "High-Performance Mobile Apps Built for Growth",
			OverviewContent: "Aplikasi smartphone kini menjadi saluran interaksi nomor satu antara bisnis dan pelanggan. TsTech mengembangkan aplikasi mobile untuk ekosistem Android dan iOS dengan fokus pada performa yang mulus (60 FPS), konsumsi baterai efisien, dan antarmuka yang ramah pengguna.\n\nKami menguasai pengembangan aplikasi cross-platform modern (Flutter & React Native) yang memungkinkan satu basis kode berkualitas tinggi berjalan di kedua platform sekaligus, memangkas biaya dan waktu peluncuran ke pasar hingga 50% tanpa mengorbankan kualitas native.",
			OverviewImage:   "https://images.unsplash.com/photo-1512941937669-90a1b58e7e9c?w=1000&auto=format&fit=crop&q=80",
			ProcessTitle:    "Mobile Development Lifecycle",
			ProcessContent:  "• **1. Discovery & App Flow**: Menyusun user flow, wireframe interaksi, dan integrasi API yang dibutuhkan.\n• **2. High-Fidelity UI/UX**: Desain visual dengan animasi transisi yang halus sesuai panduan Material Design (Android) dan Human Interface Guidelines (iOS).\n• **3. Frontend & API Integration**: Pengerjaan aplikasi mobile dengan state management modular dan sinkronisasi data realtime.\n• **4. Device Lab Testing**: Pengujian di berbagai tipe smartphone nyata (layar kecil, tablet, OS versi berbeda, performa offline mode).\n• **5. Store Submission & Launch**: Bantuan publikasi aplikasi ke Google Play Store & Apple App Store hingga disetujui (live).",
			ProcessImage:    "https://images.unsplash.com/photo-1526498460520-4c246339dccb?w=800&auto=format&fit=crop&q=80",
			TechTitle:       "Mobile & Backend Stack",
			TechContent:     "### Mobile Frameworks\n- Flutter (Dart)\n- React Native (TypeScript)\n- Swift (Native iOS)\n- Kotlin (Native Android)\n\n### Mobile Features\n- Push Notifications (Firebase FCM)\n- Offline Data Caching & SQLite\n- Biometric Login (Fingerprint / Face ID)\n- In-App Payment & QRIS Scanner\n- GPS Geolocation & Live Tracking\n\n### Backend Integration\n- RESTful API & GraphQL\n- WebSocket Real-time Messaging",
			TechImage:       "https://images.unsplash.com/photo-1551650975-87deedd944c3?w=800&auto=format&fit=crop&q=80",
			CtaTitle:        "Ingin Memiliki Aplikasi Mobile untuk Brand Anda?",
			CtaDescription:  "Diskusikan fitur dan target platform aplikasi mobile Anda dengan tim engineering kami hari ini.",
			CtaButtonText:   "Mulai Proyek Mobile App",
			CtaButtonURL:    "/konsultasi",
			SortOrder:       2,
			IsActive:        true,
			MetaTitle:       "Jasa Pembuatan Aplikasi Mobile Android & iOS | TsTech",
			MetaDescription: "Jasa pembuatan aplikasi mobile Android dan iOS menggunakan Flutter dan React Native oleh software house profesional TsTech.",
			MetaKeywords:    "jasa aplikasi android, jasa aplikasi ios, developer flutter indonesia, react native developer indonesia",
		},
		{
			Slug:            "custom-erp-information-system",
			Title:           "Custom Information Systems & ERP",
			Tagline:         "Sistem ERP, CRM, POS, HRIS, dan software manajemen operasional kustom yang dirancang presisi mengikuti alur bisnis (SOP) perusahaan Anda.",
			Badge:           "Enterprise Software",
			Icon:            "⚙️",
			OverviewTitle:   "Tailored Information Systems for Operational Excellence",
			OverviewContent: "Software siap pakai di pasaran sering kali membatasi efisiensi operasional karena memaksa perusahaan menyesuaikan proses kerja dengan software yang kaku. Solusi Custom ERP & Sistem Informasi dari TsTech dirancang khusus dari nol untuk menjawab tantangan operasional unik bisnis Anda.\n\nDari integrasi manajemen multi-gudang, pencatatan transaksi otomatis, hingga laporan keuangan laba-rugi realtime — sistem kami memangkas human error dan meningkatkan produktivitas seluruh tim.",
			OverviewImage:   "https://images.unsplash.com/photo-1454165804606-c3d57bc86b40?w=1000&auto=format&fit=crop&q=80",
			ProcessTitle:    "Enterprise Implementation Workflow",
			ProcessContent:  "• **1. Business Process Mapping**: Analisis SOP, formulir kerja, hierarki persetujuan (approval matrix), dan alur data antar divisi.\n• **2. System Architecture Design**: Perancangan skema database terenkripsi, hak akses pengguna bertingkat (Role-Based Access Control), dan modul sistem.\n• **3. Agile Module Development**: Pembangunan modul inti secara bertahap (Inventori, Penjualan, Akuntansi, SDM) dengan demo berkala.\n• **4. Data Migration & UAT**: Migrasi data lama dari Excel/sistem lawas dan pengujian penerimaan pengguna (User Acceptance Test) oleh staf Anda.\n• **5. On-Premise / Cloud Deployment & Training**: Instalasi di server perusahaan atau cloud privat, disertai dokumentasi SOP dan pelatihan intensif.",
			ProcessImage:    "https://images.unsplash.com/photo-1507679799987-c73779587ccf?w=800&auto=format&fit=crop&q=80",
			TechTitle:       "Enterprise Architecture Stack",
			TechContent:     "### Core Technologies\n- React / Next.js Admin Dashboards\n- Go Microservices / High-Throughput REST APIs\n- PostgreSQL Enterprise Database\n- Docker & Kubernetes Orchestration\n\n### Key Enterprise Modules\n- Multi-Warehouse Inventory (FIFO / Average)\n- Point of Sales (POS) & Billing Systems\n- Automated Financial Reports (Balance Sheet, P&L)\n- Role-Based Access Control (RBAC) & Audit Logs\n- Barcode / QR Code Scanner & Thermal Printing",
			TechImage:       "https://images.unsplash.com/photo-1504868584819-f8e8b4b6d7e3?w=800&auto=format&fit=crop&q=80",
			CtaTitle:        "Otomasi & Digitalisasi Operasional Bisnis Anda",
			CtaDescription:  "Diskusikan alur kerja sistem perusahaan Anda dan dapatkan blueprint arsitektur sistem dari tim konsultan TsTech.",
			CtaButtonText:   "Konsultasi Sistem ERP",
			CtaButtonURL:    "/konsultasi",
			SortOrder:       3,
			IsActive:        true,
			MetaTitle:       "Jasa Pembuatan Sistem Informasi & ERP Custom | TsTech",
			MetaDescription: "Software house penyedia jasa pembuatan software ERP kustom, CRM, POS, dan sistem manajemen perusahaan terintegrasi.",
			MetaKeywords:    "software erp custom, jasa sistem informasi manajemen, software house jakarta, aplikasi gudang custom",
		},
		{
			Slug:            "ui-ux-design-prototyping",
			Title:           "UI/UX Design & Interactive Prototyping",
			Tagline:         "Rancang antarmuka produk digital yang modern, estetis, intuitif, dan berpusat pada pengguna (user-centered) untuk memaksimalkan kepuasan dan konversi.",
			Badge:           "UI/UX Design",
			Icon:            "🎨",
			OverviewTitle:   "User-Centered Product Design & Complete Design Systems",
			OverviewContent: "Desain yang hebat bukan hanya tentang tampilan visual yang memukau, melainkan tentang bagaimana produk tersebut terasa mudah, menyenangkan, dan efisien saat digunakan oleh pelanggan. Tim UI/UX designer TsTech mengombinasikan riset mendalam perilaku pengguna, arsitektur informasi terstruktur, dan estetika visual kelas dunia.\n\nKami menyusun Design System modular di Figma yang siap dieksekusi oleh developer tanpa kebingungan (seamless developer handoff).",
			OverviewImage:   "https://images.unsplash.com/photo-1581291518857-4e27b48ff24e?w=1000&auto=format&fit=crop&q=80",
			ProcessTitle:    "Design Thinking & Prototyping Process",
			ProcessContent:  "• **1. User & Competitor Research**: Memahami target persona audiens, pain points, dan analisis benchmarking industri.\n• **2. Information Architecture & Wireframing**: Membuat sketsa kerangka struktur halaman (low-fidelity wireframes) untuk memvalidasi alur navigasi.\n• **3. Visual UI Design**: Pembuatan desain antarmuka pixel-perfect dengan palet warna harmonis, tipografi modern, dan komponen interaktif.\n• **4. Clickable Interactive Prototype**: Prototipe Figma yang dapat diklik dan disimulasikan seperti aplikasi nyata untuk pengujian pengguna.\n• **5. Design System & Handoff**: Dokumentasi token warna, font, komponen UI, dan aset siap ekspor untuk tim developer.",
			ProcessImage:    "https://images.unsplash.com/photo-1542744094-3a31f272c490?w=800&auto=format&fit=crop&q=80",
			TechTitle:       "Design Tools & Deliverables",
			TechContent:     "### Design Tools\n- Figma (Auto Layout, Variables, Component Libraries)\n- Adobe Creative Suite (Illustrator, Photoshop)\n- Whimsical & FigJam (Flowcharts & User Journeys)\n\n### Deliverables\n- Full High-Fidelity UI Mockups\n- Interactive Clickable Prototype (Figma)\n- Reusable Component Design System\n- Responsive Desktop, Tablet, & Mobile Layouts\n- Iconography & Vector Illustrations",
			TechImage:       "https://images.unsplash.com/photo-1507238691740-187a5b1d37b8?w=800&auto=format&fit=crop&q=80",
			CtaTitle:        "Wujudkan Desain Produk Digital Berkualitas Tinggi",
			CtaDescription:  "Tingkatkan konversi dan kepuasan pengguna aplikasi Anda dengan desain antarmuka modern buatan tim desainer TsTech.",
			CtaButtonText:   "Konsultasi Desain UI/UX",
			CtaButtonURL:    "/konsultasi",
			SortOrder:       4,
			IsActive:        true,
			MetaTitle:       "Jasa Desain UI/UX & Prototyping Figma | TsTech",
			MetaDescription: "Layanan desain UI/UX antarmuka website dan aplikasi mobile modern, interaktif, dan berpusat pada pengguna oleh TsTech.",
			MetaKeywords:    "jasa ui ux design, desainer figma indonesia, prototype web app, desain sistem figma",
		},
		{
			Slug:            "maintenance-cloud-support",
			Title:           "Maintenance & Cloud Infrastructure Support",
			Tagline:         "Layanan pemeliharaan teknis berkelanjutan, pembaruan keamanan, monitoring performa 24/7, dan otomatisasi backup untuk menjamin uptime maksimal website & aplikasi Anda.",
			Badge:           "Cloud & Maintenance",
			Icon:            "🔧",
			OverviewTitle:   "Reliable Engineering Support & Cloud Optimization",
			OverviewContent: "Peluncuran website atau aplikasi hanyalah awal dari siklus hidup produk digital Anda. Tanpa pemeliharaan rutin, sistem rentan terhadap celah keamanan, penurunan kecepatan muat halaman, dan gangguan server (downtime) yang merugikan bisnis.\n\nLayanan Maintenance & Cloud Support TsTech memastikan infrastruktur digital Anda selalu dalam kondisi prima, aman dari serangan siber, dan berjalan dengan kecepatan puncak.",
			OverviewImage:   "https://images.unsplash.com/photo-1558494949-ef010cbdcc31?w=1000&auto=format&fit=crop&q=80",
			ProcessTitle:    "Proactive Maintenance Protocol",
			ProcessContent:  "• **1. System & Security Audit**: Pemeriksaan menyeluruh terhadap patch sistem operasi, dependensi library, dan celah keamanan.\n• **2. 24/7 Uptime & Performance Monitoring**: Pemantauan realtime terhadap ketersediaan server, waktu respon, dan penggunaan beban CPU/RAM.\n• **3. Automated Cloud Backups**: Pencadangan data otomatis harian/mingguan ke penyimpanan cloud terpisah dengan mekanisme recovery cepat.\n• **4. Bug Fixing & Content Updates**: Penanganan segera terhadap kendala teknis dan bantuan pembaruan konten secara berkala.\n• **5. Monthly Health Report**: Pengiriman laporan berkala performa sistem, rekam jejak uptime, dan saran optimasi kapasitas.",
			ProcessImage:    "https://images.unsplash.com/photo-1451187580459-43490279c0fa?w=800&auto=format&fit=crop&q=80",
			TechTitle:       "Cloud Technologies & Tools",
			TechContent:     "### Cloud Providers & Platforms\n- Amazon Web Services (AWS)\n- Google Cloud Platform (GCP)\n- DigitalOcean & Linode VPS\n- Cloudflare CDN, DNS, & DDoS Protection\n\n### Monitoring & DevOps\n- Prometheus & Grafana Metrics\n- Sentry Error Tracking & Alerts\n- Automated SSL Certificate Renewal\n- Offsite S3 Encrypted Database Backups",
			TechImage:       "https://images.unsplash.com/photo-1544197150-b99a580bb7a8?w=800&auto=format&fit=crop&q=80",
			CtaTitle:        "Lindungi & Optimalkan Sistem Digital Anda",
			CtaDescription:  "Serahkan urusan teknis dan pemeliharaan server kepada tim ahli TsTech agar Anda dapat fokus mengembangkan bisnis.",
			CtaButtonText:   "Amankan Sistem Anda",
			CtaButtonURL:    "/konsultasi",
			SortOrder:       5,
			IsActive:        true,
			MetaTitle:       "Jasa Maintenance Website, Aplikasi & Cloud Server | TsTech",
			MetaDescription: "Layanan pemeliharaan sistem, monitoring server 24/7, security audit, dan cloud optimization untuk kestabilan bisnis Anda.",
			MetaKeywords:    "jasa maintenance website, pemeliharaan aplikasi, cloud support indonesia, server monitoring",
		},
	}

	for _, s := range services {
		db.Create(&s)
	}
	log.Println("🚀 Seeded default services with detailed sections")
}

func seedProducts(db *gorm.DB) {
	var count int64
	db.Model(&model.Product{}).Count(&count)
	if count > 0 {
		return
	}

	products := []model.Product{
		{
			Slug:            "pos-resto-multi-branch",
			Title:           "TsTech POS & Resto Multi-Branch Cloud",
			Tagline:         "Sistem Kasir & Manajemen Restoran, Kafe, dan F&B Berbasis Cloud Terintegrasi Multi-Outlet",
			Badge:           "SaaS & Ready Product",
			Category:        "Point of Sale",
			Icon:            "🍽️",
			Thumbnail:       "https://images.unsplash.com/photo-1555396273-367ea4eb4db5?w=800&auto=format&fit=crop&q=80",
			Images:          `["https://images.unsplash.com/photo-1555396273-367ea4eb4db5?w=1000&auto=format&fit=crop&q=80","https://images.unsplash.com/photo-1556742049-0a67e55722c0?w=1000&auto=format&fit=crop&q=80","https://images.unsplash.com/photo-1508873696983-2df5703bc20d?w=1000&auto=format&fit=crop&q=80"]`,
			Price:           3500000,
			PriceDiscount:   4900000,
			PriceType:       "one_time",
			DemoURL:         "https://demo-resto.kotban.com",
			DocURL:          "https://docs.kotban.com/pos-resto",
			Features:        "• **Kitchen Display System (KDS)**: Tiket pesanan otomatis terkirim langsung ke layar dapur realtime.\n• **Multi-Outlet Synchronization**: Kelola menu, harga, dan stok dari ratusan cabang dalam satu dashboard pusat.\n• **QR Table Ordering**: Pelanggan scan barcode di meja, pesan menu, dan bayar mandiri tanpa antre.\n• **Payment Gateway QRIS & VA**: Mendukung pembayaran tunai, QRIS Dinamis, GoPay, OVO, ShopeePay, dan Transfer Bank.\n• **Inventori Resep & HPP (COGS)**: Pengurangan stok bahan baku otomatis berdasarkan takaran menu terjual.\n• **Laporan Penjualan & Laba/Rugi**: Rekapitulasi omset harian, shift kasir, split bill, dan analisa menu terlaris.",
			TechStack:       "Next.js 15, Go (Echo), PostgreSQL, TailwindCSS, WebSocket, Redis",
			Overview:        "### Solusi Kasir Digital Modern untuk Pertumbuhan Bisnis F&B Anda\n\nTsTech POS & Resto dirancang khusus untuk memenuhi kebutuhan operasional restoran cepat saji, coffee shop, bakery, hingga fine dining dengan banyak cabang. Dengan arsitektur hybrid online-offline, transaksi tetap berjalan lancar meski koneksi internet terputus sesaat.\n\n### Keunggulan Utama:\n- **Tanpa Biaya Langganan Bulanan**: Dapatkan source code penuh dan deploy di server milik Anda sendiri (100% Hak Milik).\n- **Kecepatan Transaksi Tinggi**: UI didesain sangat intuitif untuk kasir, memproses pesanan dalam hitungan detik.\n- **Support Hardware Kasir Lengkap**: Kompatibel dengan printer thermal Bluetooth/USB, cash drawer, dan barcode scanner.",
			IsFeatured:      true,
			IsActive:        true,
			SortOrder:       1,
			MetaTitle:       "Software Kasir Restoran & Kafe Multi-Cabang | TsTech POS Resto",
			MetaDescription: "Software POS dan manajemen resto berbasis cloud lengkap dengan KDS, QR Dine-In, stok bahan baku, dan integrasi QRIS.",
			MetaKeywords:    "software kasir resto, aplikasi pos restoran, pos cloud cafe, aplikasi fnb multi cabang",
		},
		{
			Slug:            "erp-core-enterprise",
			Title:           "TsTech ERP Core Enterprise Edition",
			Tagline:         "Software Manajemen Stok Multi-Gudang, Akuntansi, Purchasing, dan Distribusi Penjualan",
			Badge:           "Enterprise Solution",
			Category:        "ERP & Keuangan",
			Icon:            "🏢",
			Thumbnail:       "https://images.unsplash.com/photo-1460925895917-afdab827c52f?w=800&auto=format&fit=crop&q=80",
			Images:          `["https://images.unsplash.com/photo-1460925895917-afdab827c52f?w=1000&auto=format&fit=crop&q=80","https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=1000&auto=format&fit=crop&q=80"]`,
			Price:           9500000,
			PriceDiscount:   14000000,
			PriceType:       "one_time",
			DemoURL:         "https://demo-erp.kotban.com",
			DocURL:          "https://docs.kotban.com/erp-core",
			Features:        "• **Multi-Warehouse & Bin Location**: Pelacakan stok realtime di berbagai gudang, nomor batch, dan expiry date.\n• **Akuntansi Standar PSAK**: Jurnal otomatis, buku besar, neraca saldo, laporan laba rugi, dan arus kas otomatis.\n• **Procurement & Purchase Approval**: Alur pengajuan PO bertingkat dengan validasi limit anggaran departemen.\n• **Sales Order & Invoicing**: Penerbitan penawaran harga, surat jalan, faktur pajak PPN 11%, dan kuitansi pelunasan.\n• **Role-Based Access Control (RBAC)**: Pembatasan hak akses detail per divisi (Gudang, Sales, Finance, Direksi).\n• **Audit Trail & Log Lengkap**: Rekam jejak seluruh aktivitas manipulasi data untuk mencegah fraud internal.",
			TechStack:       "React, Go, PostgreSQL, Redis, Docker, TailwindCSS",
			Overview:        "### Otomatisasi Alur Kerja Bisnis Menyeluruh Tanpa Batasan Lisensi User\n\nTsTech ERP Core Enterprise menjembatani divisi gudang, keuangan, pengadaan, dan penjualan ke dalam satu platform terpadu. Dibuat dengan clean architecture Go dan PostgreSQL berkinerja tinggi, sistem sanggup menangani jutaan baris transaksi tanpa lag.\n\n### Mengapa Memilih TsTech ERP?\n1. **Unlimited Users & Unlimited Data**: Tidak ada batasan jumlah staf atau jumlah cabang.\n2. **Customizable**: Mudah disesuaikan dengan SOP khusus perusahaan Anda.\n3. **Export Laporan Lengkap**: Ekspor Excel, PDF, dan API integrasi ke sistem pihak ketiga.",
			IsFeatured:      true,
			IsActive:        true,
			SortOrder:       2,
			MetaTitle:       "Software ERP Manajemen Stok & Akuntansi Perusahaan | TsTech ERP",
			MetaDescription: "Software ERP custom Indonesia untuk manajemen pergudangan, keuangan PSAK, sales order, dan procurement tanpa biaya langganan bulanan.",
			MetaKeywords:    "software erp indonesia, aplikasi stok gudang, software akuntansi enterprise, custom erp go",
		},
		{
			Slug:            "klinik-emr-satusehat",
			Title:           "TsTech Medika & Clinic EMR (SatuSehat Ready)",
			Tagline:         "Aplikasi Rekam Medis Elektronik (RME) Standar Kemenkes, Antrean Pasien & Kasir Apotek",
			Badge:           "Kemenkes SatuSehat Ready",
			Category:        "Klinik & Medika",
			Icon:            "🏥",
			Thumbnail:       "https://images.unsplash.com/photo-1576091160399-112ba8d25d1d?w=800&auto=format&fit=crop&q=80",
			Images:          `["https://images.unsplash.com/photo-1576091160399-112ba8d25d1d?w=1000&auto=format&fit=crop&q=80","https://images.unsplash.com/photo-1519494026892-80bbd2d6fd0d?w=1000&auto=format&fit=crop&q=80"]`,
			Price:           6500000,
			PriceDiscount:   8900000,
			PriceType:       "one_time",
			DemoURL:         "https://demo-medika.kotban.com",
			DocURL:          "https://docs.kotban.com/medika-emr",
			Features:        "• **Standar SatuSehat Kemenkes**: Format data FHIR terstandarisasi siap sinkronisasi ke platform SatuSehat.\n• **Rekam Medis Elektronik (SOAP)**: Input diagnosa ICD-10, tindakan ICD-9-CM, odontogram gigi, dan riwayat alergi.\n• **Sistem Antrean Poliklinik & Display Suara**: Panggilan nomor antrean otomatis dengan audio dan monitor TV ruang tunggu.\n• **Manajemen Apotek & Stok Obat**: Pengurangan obat otomatis dari e-resep dokter, kartu stok, dan peringatan expired.\n• **Billing & Pembayaran Kasir**: Cetak nota pembayaran, kuitansi tindakan medis, dan integrasi pembayaran QRIS.",
			TechStack:       "Next.js 15, Go (Echo), PostgreSQL, TailwindCSS",
			Overview:        "### Digitalisasi Klinik Anda Sesuai Regulasi Permenkes No. 24 Tahun 2022\n\nTsTech Medika EMR membantu klinik pratama, klinik utama, dan tempat praktik mandiri dokter mengelola pendaftaran pasien, pencatatan rekam medis elektronik, antrean, hingga kasir apotek dengan cepat, akurat, dan aman.\n\n### Fitur Utama:\n- Enkripsi Data Medis Tingkat Tinggi\n- E-Prescription & Label Obat Otomatis\n- Manajemen Jadwal Praktik Dokter & Kuota Pasien",
			IsFeatured:      true,
			IsActive:        true,
			SortOrder:       3,
			MetaTitle:       "Software Rekam Medis Elektronik Klinik SatuSehat | TsTech Medika",
			MetaDescription: "Software RME klinik terintegrasi SatuSehat Kemenkes, antrean poli, kasir apotek, dan manajemen jadwal dokter.",
			MetaKeywords:    "software klinik satusehat, aplikasi rme klinik, rekam medis elektronik, software apotek klinik",
		},
		{
			Slug:            "smart-school-lms-cbt",
			Title:           "Schola LMS & Sistem Informasi Manajemen Sekolah",
			Tagline:         "Platform E-Learning Interaktif, Bank Soal CBT Anti-Curang, Administrasi SPP & Rapor Digital Terpadu",
			Badge:           "Popular for Education & SaaS",
			Category:        "Sekolah & LMS",
			Icon:            "🏫",
			Thumbnail:       "https://images.unsplash.com/photo-1580582932707-520aed937b7b?w=800&auto=format&fit=crop&q=80",
			Images:          `["https://images.unsplash.com/photo-1580582932707-520aed937b7b?w=1000&auto=format&fit=crop&q=80","https://images.unsplash.com/photo-1509062522246-3755977927d7?w=1000&auto=format&fit=crop&q=80"]`,
			Price:           4500000,
			PriceDiscount:   6500000,
			PriceType:       "one_time",
			DemoURL:         "https://demo.schola.tstech.id",
			DocURL:          "https://docs.tstech.id/schola",
			Features:        "• **Subdomain Sekolah Instan**: Contoh: `smkn1.schola.tstech.id` langsung aktif otomatis.\n• **E-Rapor Kurikulum Merdeka & K13**: Cetak rapor format resmi Kemendikbudristek sekali klik.\n• **CBT (Computer Based Test)**: Ujian online anti-curang dengan browser lockdown dan bank soal fleksibel.\n• **Billing & Notifikasi Tagihan SPP WhatsApp**: Orang tua menerima slip rincian SPP dan bayar via QRIS/VA otomatis.\n• **Portal Guru, Siswa & Wali Murid**: Akses nilai, jadwal mengajar, materi LMS, dan absensi harian realtime.",
			TechStack:       "Next.js 15, Go (Echo), PostgreSQL, Cloudflare S3",
			Overview:        "### Solusi Digital Terpadu untuk Sekolah Modern & Pesantren\n\nTsTech School LMS memudahkan guru membuat bahan ajar dan ujian online yang aman dari kecurangan, sekaligus memberikan kemudahan bagi manajemen sekolah dalam mengelola administrasi keuangan SPP dan rekapitulasi nilai.",
			IsFeatured:      true,
			IsActive:        true,
			SortOrder:       4,
			MetaTitle:       "Aplikasi Smart School & LMS CBT Ujian Online | TsTech School",
			MetaDescription: "Sistem informasi akademik sekolah, CBT ujian online anti curang, pembayaran SPP QRIS, dan e-rapor kurikulum merdeka.",
			MetaKeywords:    "aplikasi sekolah lms, software cbt ujian sekolah, sistem informasi akademik, aplikasi pembayaran spp",
			IsSaaS:          true,
			SubdomainPattern: "{tenant}.schola.tstech.id",
			BaseDomain:       "schola.tstech.id",
			APISecretKey:     "sec_schola_live_89123891",
		},
		{
			Slug:            "pos-resto-multi-branch",
			Title:           "TsTech POS & Resto Multi-Branch Cloud",
			Tagline:         "Sistem Kasir & Manajemen Restoran, Kafe, dan F&B Berbasis Cloud Terintegrasi Multi-Outlet",
			Badge:           "SaaS & Ready Product",
			Category:        "Point of Sale",
			Icon:            "🍽️",
			Thumbnail:       "https://images.unsplash.com/photo-1555396273-367ea4eb4db5?w=800&auto=format&fit=crop&q=80",
			Images:          `["https://images.unsplash.com/photo-1555396273-367ea4eb4db5?w=1000&auto=format&fit=crop&q=80","https://images.unsplash.com/photo-1556742049-0a67e55722c0?w=1000&auto=format&fit=crop&q=80","https://images.unsplash.com/photo-1508873696983-2df5703bc20d?w=1000&auto=format&fit=crop&q=80"]`,
			Price:           3500000,
			PriceDiscount:   4900000,
			PriceType:       "one_time",
			DemoURL:         "https://demo.restopos.tstech.id",
			DocURL:          "https://docs.kotban.com/pos-resto",
			Features:        "• **Kitchen Display System (KDS)**: Tiket pesanan otomatis terkirim langsung ke layar dapur realtime.\n• **Multi-Outlet Synchronization**: Kelola menu, harga, dan stok dari ratusan cabang dalam satu dashboard pusat.\n• **QR Table Ordering**: Pelanggan scan barcode di meja, pesan menu, dan bayar mandiri tanpa antre.\n• **Payment Gateway QRIS & VA**: Mendukung pembayaran tunai, QRIS Dinamis, GoPay, OVO, ShopeePay, dan Transfer Bank.\n• **Inventori Resep & HPP (COGS)**: Pengurangan stok bahan baku otomatis berdasarkan takaran menu terjual.\n• **Laporan Penjualan & Laba/Rugi**: Rekapitulasi omset harian, shift kasir, split bill, dan analisa menu terlaris.",
			TechStack:       "Next.js 15, Go (Echo), PostgreSQL, TailwindCSS, WebSocket, Redis",
			Overview:        "### Solusi Kasir Digital Modern untuk Pertumbuhan Bisnis F&B Anda\n\nTsTech POS & Resto dirancang khusus untuk memenuhi kebutuhan operasional restoran cepat saji, coffee shop, bakery, hingga fine dining dengan banyak cabang. Dengan arsitektur hybrid online-offline, transaksi tetap berjalan lancar meski koneksi internet terputus sesaat.\n\n### Keunggulan Utama:\n- **Tanpa Biaya Langganan Bulanan**: Dapatkan source code penuh dan deploy di server milik Anda sendiri (100% Hak Milik).\n- **Kecepatan Transaksi Tinggi**: UI didesain sangat intuitif untuk kasir, memproses pesanan dalam hitungan detik.\n- **Support Hardware Kasir Lengkap**: Kompatibel dengan printer thermal Bluetooth/USB, cash drawer, dan barcode scanner.",
			IsFeatured:      true,
			IsActive:        true,
			SortOrder:       1,
			MetaTitle:       "Software Kasir Restoran & Kafe Multi-Cabang | TsTech POS Resto",
			MetaDescription: "Software POS dan manajemen resto berbasis cloud lengkap dengan KDS, QR Dine-In, stok bahan baku, dan integrasi QRIS.",
			MetaKeywords:    "software kasir resto, aplikasi pos restoran, pos cloud cafe, aplikasi fnb multi cabang",
			IsSaaS:          true,
			SubdomainPattern: "{tenant}.restopos.tstech.id",
			BaseDomain:       "restopos.tstech.id",
			APISecretKey:     "sec_restopos_live_72918231",
		},
		{
			Slug:            "klinik-emr-satusehat",
			Title:           "TsTech Medika & Clinic EMR (SatuSehat Ready)",
			Tagline:         "Aplikasi Rekam Medis Elektronik (RME) Standar Kemenkes, Antrean Pasien & Kasir Apotek",
			Badge:           "Kemenkes SatuSehat Ready",
			Category:        "Klinik & Medika",
			Icon:            "🏥",
			Thumbnail:       "https://images.unsplash.com/photo-1576091160399-112ba8d25d1d?w=800&auto=format&fit=crop&q=80",
			Images:          `["https://images.unsplash.com/photo-1576091160399-112ba8d25d1d?w=1000&auto=format&fit=crop&q=80","https://images.unsplash.com/photo-1519494026892-80bbd2d6fd0d?w=1000&auto=format&fit=crop&q=80"]`,
			Price:           6500000,
			PriceDiscount:   8900000,
			PriceType:       "one_time",
			DemoURL:         "https://demo.medika.tstech.id",
			DocURL:          "https://docs.kotban.com/medika-emr",
			Features:        "• **Standar SatuSehat Kemenkes**: Format data FHIR terstandarisasi siap sinkronisasi ke platform SatuSehat.\n• **Rekam Medis Elektronik (SOAP)**: Input diagnosa ICD-10, tindakan ICD-9-CM, odontogram gigi, dan riwayat alergi.\n• **Sistem Antrean Poliklinik & Display Suara**: Panggilan nomor antrean otomatis dengan audio dan monitor TV ruang tunggu.\n• **Manajemen Apotek & Stok Obat**: Pengurangan obat otomatis dari e-resep dokter, kartu stok, dan peringatan expired.\n• **Billing & Pembayaran Kasir**: Cetak nota pembayaran, kuitansi tindakan medis, dan integrasi pembayaran QRIS.",
			TechStack:       "Next.js 15, Go (Echo), PostgreSQL, TailwindCSS",
			Overview:        "### Digitalisasi Klinik Anda Sesuai Regulasi Permenkes No. 24 Tahun 2022\n\nTsTech Medika EMR membantu klinik pratama, klinik utama, dan tempat praktik mandiri dokter mengelola pendaftaran pasien, pencatatan rekam medis elektronik, antrean, hingga kasir apotek dengan cepat, akurat, dan aman.\n\n### Fitur Utama:\n- Enkripsi Data Medis Tingkat Tinggi\n- E-Prescription & Label Obat Otomatis\n- Manajemen Jadwal Praktik Dokter & Kuota Pasien",
			IsFeatured:      true,
			IsActive:        true,
			SortOrder:       3,
			MetaTitle:       "Software Rekam Medis Elektronik Klinik SatuSehat | TsTech Medika",
			MetaDescription: "Software RME klinik terintegrasi SatuSehat Kemenkes, antrean poli, kasir apotek, dan manajemen jadwal dokter.",
			MetaKeywords:    "software klinik satusehat, aplikasi rme klinik, rekam medis elektronik, software apotek klinik",
			IsSaaS:          true,
			SubdomainPattern: "{tenant}.medika.tstech.id",
			BaseDomain:       "medika.tstech.id",
			APISecretKey:     "sec_medika_live_38127391",
		},
		{
			Slug:            "erp-core-enterprise",
			Title:           "TsTech ERP Core Enterprise Edition",
			Tagline:         "Software Manajemen Stok Multi-Gudang, Akuntansi, Purchasing, dan Distribusi Penjualan",
			Badge:           "Enterprise Solution",
			Category:        "ERP & Keuangan",
			Icon:            "🏢",
			Thumbnail:       "https://images.unsplash.com/photo-1460925895917-afdab827c52f?w=800&auto=format&fit=crop&q=80",
			Images:          `["https://images.unsplash.com/photo-1460925895917-afdab827c52f?w=1000&auto=format&fit=crop&q=80","https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=1000&auto=format&fit=crop&q=80"]`,
			Price:           9500000,
			PriceDiscount:   14000000,
			PriceType:       "one_time",
			DemoURL:         "https://demo-erp.kotban.com",
			DocURL:          "https://docs.kotban.com/erp-core",
			Features:        "• **Multi-Warehouse & Bin Location**: Pelacakan stok realtime di berbagai gudang, nomor batch, dan expiry date.\n• **Akuntansi Standar PSAK**: Jurnal otomatis, buku besar, neraca saldo, laporan laba rugi, dan arus kas otomatis.\n• **Procurement & Purchase Approval**: Alur pengajuan PO bertingkat dengan validasi limit anggaran departemen.\n• **Sales Order & Invoicing**: Penerbitan penawaran harga, surat jalan, faktur pajak PPN 11%, dan kuitansi pelunasan.\n• **Role-Based Access Control (RBAC)**: Pembatasan hak akses detail per divisi (Gudang, Sales, Finance, Direksi).\n• **Audit Trail & Log Lengkap**: Rekam jejak seluruh aktivitas manipulasi data untuk mencegah fraud internal.",
			TechStack:       "React, Go, PostgreSQL, Redis, Docker, TailwindCSS",
			Overview:        "### Otomatisasi Alur Kerja Bisnis Menyeluruh Tanpa Batasan Lisensi User\n\nTsTech ERP Core Enterprise menjembatani divisi gudang, keuangan, pengadaan, dan penjualan ke dalam satu platform terpadu. Dibuat dengan clean architecture Go dan PostgreSQL berkinerja tinggi, sistem sanggup menangani jutaan baris transaksi tanpa lag.\n\n### Mengapa Memilih TsTech ERP?\n1. **Unlimited Users & Unlimited Data**: Tidak ada batasan jumlah staf atau jumlah cabang.\n2. **Customizable**: Mudah disesuaikan dengan SOP khusus perusahaan Anda.\n3. **Export Laporan Lengkap**: Ekspor Excel, PDF, dan API integrasi ke sistem pihak ketiga.",
			IsFeatured:      true,
			IsActive:        true,
			SortOrder:       2,
			MetaTitle:       "Software ERP Manajemen Stok & Akuntansi Perusahaan | TsTech ERP",
			MetaDescription: "Software ERP custom Indonesia untuk manajemen pergudangan, keuangan PSAK, sales order, dan procurement tanpa biaya langganan bulanan.",
			MetaKeywords:    "software erp indonesia, aplikasi stok gudang, software akuntansi enterprise, custom erp go",
		},
		{
			Slug:            "b2b-wholesale-ecommerce",
			Title:           "TsTech B2B Commerce & Wholesale Platform",
			Tagline:         "Platform E-Commerce B2B Grosir, Tiered Pricing, Integrasi Kargo Logistik & Distributor Portal",
			Badge:           "High Scalability",
			Category:        "E-Commerce",
			Icon:            "🛒",
			Thumbnail:       "https://images.unsplash.com/photo-1556742049-0a67e55722c0?w=800&auto=format&fit=crop&q=80",
			Images:          `["https://images.unsplash.com/photo-1556742049-0a67e55722c0?w=1000&auto=format&fit=crop&q=80","https://images.unsplash.com/photo-1472851294608-062f824d29cc?w=1000&auto=format&fit=crop&q=80"]`,
			Price:           7500000,
			PriceDiscount:   11000000,
			PriceType:       "one_time",
			DemoURL:         "https://demo-b2b.kotban.com",
			DocURL:          "https://docs.kotban.com/b2b-commerce",
			Features:        "• **Tiered Pricing & Minimum Order Quantity (MOQ)**: Harga khusus untuk Reseller, Agen, dan Distributor Utama.\n• **Request for Quotation (RFQ)**: Fitur negosiasi harga partai besar langsung dalam antarmuka web.\n• **Integrasi Ekspedisi Kargo & Ongkir Otomatis**: Menghitung tarif pengiriman kargo darat/laut/udara otomatis.\n• **Term of Payment (TOP / Tempo)**: Manajemen plafon kredit dan jatuh tempo penagihan piutang distributor.\n• **Pajak Faktur PPN & PPh 22**: Otomatisasi penerbitan e-faktur sesuai regulasi perpajakan B2B.",
			TechStack:       "Next.js 15, Go, PostgreSQL, Redis, Elasticsearch",
			Overview:        "### Tingkatkan Penjualan Grosir dan Distribusi Produk Skala Nasional\n\nPlatform B2B Commerce TsTech dirancang untuk produsen, distributor, dan pemilik brand yang ingin mendigitalkan jalur pemesanan agen dan grosir secara transparan dan efisien.",
			IsFeatured:      true,
			IsActive:        true,
			SortOrder:       5,
			MetaTitle:       "Platform E-Commerce B2B Grosir & Distributor | TsTech B2B",
			MetaDescription: "Platform marketplace B2B dan e-commerce grosir dengan harga bertingkat, tempo pembayaran, dan kalkulasi ongkir kargo.",
			MetaKeywords:    "ecommerce b2b indonesia, platform grosir distributor, software b2b marketplace",
		},
		{
			Slug:            "hr-payroll-face-recognition",
			Title:           "TsTech HR & Payroll Pro (Face Recognition & GPS)",
			Tagline:         "Aplikasi Absensi Geolocation Anti-Fake GPS, Perhitungan Gaji PPh 21 TER, Cuti & KPI",
			Badge:           "Best Seller",
			Category:        "HRM & Payroll",
			Icon:            "💼",
			Thumbnail:       "https://images.unsplash.com/photo-1522071820081-009f0129c71c?w=800&auto=format&fit=crop&q=80",
			Images:          `["https://images.unsplash.com/photo-1522071820081-009f0129c71c?w=1000&auto=format&fit=crop&q=80","https://images.unsplash.com/photo-1600880292203-757bb62b4baf?w=1000&auto=format&fit=crop&q=80"]`,
			Price:           5000000,
			PriceDiscount:   7500000,
			PriceType:       "one_time",
			DemoURL:         "https://demo-hr.kotban.com",
			DocURL:          "https://docs.kotban.com/hr-payroll",
			Features:        "• **Absensi Wajah & Radius Geofencing**: Deteksi biometrik wajah dan verifikasi lokasi GPS anti fake GPS / mock location.\n• **Kalkulator Gaji Otomatis (PPh 21 TER & BPJS)**: Perhitungan pajak tarif efektif rata-rata (TER) dan iuran BPJS TK & Kesehatan.\n• **Pengajuan Cuti, Izin & Reimbursement**: Approval bertingkat langsung dari smartphone atasan secara realtime.\n• **Distribusi Slip Gaji PDF & WhatsApp**: Kirim slip gaji digital terenkripsi password langsung ke WhatsApp karyawan.\n• **Manajemen Shift Kerja Fleksibel**: Support sistem kerja 3 shift, lembur (overtime), dan tukar jadwal jaga.",
			TechStack:       "React, Go, Flutter (Mobile Android & iOS), PostgreSQL",
			Overview:        "### Kelola Manajemen Karyawan dan Payroll Tanpa Kerumitan Manual\n\nTsTech HR & Payroll Pro menghemat waktu tim HRD hingga 80% dalam merekap absensi dan menghitung penggajian bulanan.",
			IsFeatured:      true,
			IsActive:        true,
			SortOrder:       6,
			MetaTitle:       "Aplikasi Absensi GPS Wajah & Payroll PPh 21 TER | TsTech HR",
			MetaDescription: "Aplikasi HRIS absensi GPS selfie, perhitungan gaji PPh 21 TER otomatis, cuti online, dan slip gaji WhatsApp.",
			MetaKeywords:    "aplikasi absensi online gps, software payroll pph 21, aplikasi hris indonesia, slip gaji whatsapp",
		},
	}

	for _, p := range products {
		var existing model.Product
		if err := db.Where("slug = ?", p.Slug).First(&existing).Error; err != nil {
			db.Create(&p)
		} else {
			existing.IsSaaS = p.IsSaaS
			existing.SubdomainPattern = p.SubdomainPattern
			existing.BaseDomain = p.BaseDomain
			if p.APISecretKey != "" && existing.APISecretKey == "" {
				existing.APISecretKey = p.APISecretKey
			}
			db.Save(&existing)
		}
	}

	// Link SaaS plans to products by slug matching
	var allProducts []model.Product
	db.Find(&allProducts)
	for _, prod := range allProducts {
		if prod.IsSaaS {
			var saasProd model.SaaSProduct
			if err := db.Where("slug = ?", prod.Slug).First(&saasProd).Error; err == nil {
				db.Model(&model.SaaSPlan{}).Where("saa_s_product_id = ?", saasProd.ID).Update("product_id", prod.ID)
			}
		}
	}

	log.Println("🛍️ Seeded and synchronized unified ready-to-sell and SaaS products")
}

func seedSaaSProducts(db *gorm.DB) {
	var count int64
	db.Model(&model.SaaSProduct{}).Count(&count)
	if count > 0 {
		return
	}

	saasItems := []struct {
		Product model.SaaSProduct
		Plans   []model.SaaSPlan
	}{
		{
			Product: model.SaaSProduct{
				Slug:             "schola",
				Name:             "Schola LMS & Sistem Informasi Manajemen Sekolah",
				Tagline:          "SaaS Manajemen Sekolah Terpadu: PPDB Online, Absensi RFID/WhatsApp, E-Rapor Kurikulum Merdeka, CBT Ujian & SPP Terintegrasi",
				Category:         "Pendidikan & Sekolah",
				Icon:             "🏫",
				SubdomainPattern: "{tenant}.schola.tstech.id",
				BaseDomain:       "schola.tstech.id",
				DemoURL:          "https://demo.schola.tstech.id",
				DocURL:           "https://docs.tstech.id/schola",
				Thumbnail:        "https://images.unsplash.com/photo-1580582932707-520aed937b7b?w=800&auto=format&fit=crop&q=80",
				Images:           `["https://images.unsplash.com/photo-1580582932707-520aed937b7b?w=1000&auto=format&fit=crop&q=80"]`,
				Features:         "• **Subdomain Sekolah Instan**: Contoh: `smkn1.schola.tstech.id` langsung aktif otomatis.\n• **E-Rapor Kurikulum Merdeka & K13**: Cetak rapor format resmi Kemendikbudristek sekali klik.\n• **CBT (Computer Based Test)**: Ujian online anti-curang dengan browser lockdown dan bank soal fleksibel.\n• **Billing & Notifikasi Tagihan SPP WhatsApp**: Orang tua menerima slip rincian SPP dan bayar via QRIS/VA otomatis.\n• **Portal Guru, Siswa & Wali Murid**: Akses nilai, jadwal mengajar, materi LMS, dan absensi harian realtime.",
				TechStack:        "Next.js 15, Go Fiber, PostgreSQL Multi-Tenancy, Redis, WebSocket",
				Overview:         "### Solusi Digitalisasi Sekolah Modern Terlengkap\n\nSchola dirancang untuk mempermudah operasional sekolah SD, SMP, SMA, SMK hingga Pesantren dan Lembaga Kursus dengan sistem multi-tenant terisolasi yang aman dan cepat.",
				IsActive:         true,
				SortOrder:        1,
			},
			Plans: []model.SaaSPlan{
				{
					Name:         "Starter (Bulanan)",
					Code:         "schola_starter_monthly",
					Interval:     "monthly",
					PriceMonthly: 350000,
					PriceYearly:  3500000,
					DiscountPct:  0,
					Features:     `["Maksimal 300 Siswa Aktif","E-Rapor Kurikulum Merdeka","CBT Ujian Online (500 user concurrent)","Manajemen SPP & Kas Sekolah","Notifikasi WhatsApp Terintegrasi","Subdomain gratis namasekolah.schola.tstech.id"]`,
					MaxUsers:     300,
					MaxStorageGB: 10,
					IsPopular:    false,
					IsActive:     true,
					SortOrder:    1,
				},
				{
					Name:         "Professional (Tahunan - Hemat 2 Bulan)",
					Code:         "schola_pro_yearly",
					Interval:     "yearly",
					PriceMonthly: 300000,
					PriceYearly:  3000000,
					DiscountPct:  15,
					Features:     `["Maksimal 1.000 Siswa Aktif","Semua Fitur Starter","PPDB Online dengan Form Custom","Absensi RFID & Notifikasi WhatsApp Realtime","Prioritas Bantuan CS 24/7","Bisa pasang Custom Domain (lms.sekolahanda.sch.id)"]`,
					MaxUsers:     1000,
					MaxStorageGB: 50,
					IsPopular:    true,
					IsActive:     true,
					SortOrder:    2,
				},
				{
					Name:         "Enterprise / Kampus (Tahunan)",
					Code:         "schola_enterprise_yearly",
					Interval:     "yearly",
					PriceMonthly: 750000,
					PriceYearly:  7500000,
					DiscountPct:  20,
					Features:     `["Siswa Unlimited (Tak Terbatas)","Semua Fitur Professional","Server Dedicated Isolation","Integrasi Fingerprint & Gerbang Turnstile","Custom Modul & On-Premise Training","Garansi SLA 99.9% Uptime"]`,
					MaxUsers:     0,
					MaxStorageGB: 200,
					IsPopular:    false,
					IsActive:     true,
					SortOrder:    3,
				},
			},
		},
		{
			Product: model.SaaSProduct{
				Slug:             "restopos",
				Name:             "TsTech RestoPOS & Cloud Kitchen",
				Tagline:          "SaaS Point of Sales Kasir Restoran, Kitchen Display System (KDS), Order Meja QR & Manajemen Stok Bahan Baku Realtime",
				Category:         "FnB & Retail",
				Icon:             "🍽️",
				SubdomainPattern: "{tenant}.restopos.tstech.id",
				BaseDomain:       "restopos.tstech.id",
				DemoURL:          "https://demo.restopos.tstech.id",
				DocURL:           "https://docs.tstech.id/restopos",
				Thumbnail:        "https://images.unsplash.com/photo-1555396273-367ea4eb4db5?w=800&auto=format&fit=crop&q=80",
				Images:           `["https://images.unsplash.com/photo-1555396273-367ea4eb4db5?w=1000&auto=format&fit=crop&q=80"]`,
				Features:         "• **Subdomain Kafe/Resto Instan**: Akses kasir dari `namaresto.restopos.tstech.id`.\n• **QR Menu & Self Order Meja**: Pelanggan scan QR meja, pilih menu, dan bayar mandiri.\n• **Kitchen Display Screen (KDS)**: Pesanan kasir langsung muncul di layar dapur realtime.\n• **HPP & Manajemen Resep Bahan Baku**: Potong stok bahan otomatis saat menu terjual.\n• **Support Printer Thermal Bluetooth & LAN**: Cetak struk kasir dan checker dapur tanpa jeda.",
				TechStack:        "Next.js 15, Go, PostgreSQL, Redis Pub/Sub",
				Overview:         "### Percepat Layanan dan Cegah Kebocoran Stok Restoran Anda\n\nRestoPOS menghadirkan automasi restoran kelas enterprise untuk pemilik kedai kopi, kafe modern, restoran cepat saji, hingga cloud kitchen multi-cabang.",
				IsActive:         true,
				SortOrder:        2,
			},
			Plans: []model.SaaSPlan{
				{
					Name:         "Single Outlet (Bulanan)",
					Code:         "restopos_single_monthly",
					Interval:     "monthly",
					PriceMonthly: 199000,
					PriceYearly:  1990000,
					DiscountPct:  0,
					Features:     `["1 Outlet / Cabang","Kasir Kasir POS Tablet/PC Tak Terbatas","Order QR Menu Meja","Laporan Penjualan & Laba Rugi Harian","Manajemen Inventori & Resep"]`,
					MaxUsers:     10,
					MaxStorageGB: 5,
					IsPopular:    false,
					IsActive:     true,
					SortOrder:    1,
				},
				{
					Name:         "Multi-Outlet Pro (Bulanan)",
					Code:         "restopos_multi_monthly",
					Interval:     "monthly",
					PriceMonthly: 449000,
					PriceYearly:  4490000,
					DiscountPct:  15,
					Features:     `["Hingga 3 Outlet Cabang","Semua Fitur Single Outlet","Kitchen Display System (KDS)","Transfer Stok Antar Cabang","Hak Akses Supervisor & Manajer"]`,
					MaxUsers:     30,
					MaxStorageGB: 20,
					IsPopular:    true,
					IsActive:     true,
					SortOrder:    2,
				},
			},
		},
		{
			Product: model.SaaSProduct{
				Slug:             "medika",
				Name:             "TsTech Medika EMR & Klinik Pintar",
				Tagline:          "SaaS Rekam Medis Elektronik (RME) Standar SatuSehat Kemenkes RI, Antrean Online Pasien & Farmasi Apotek",
				Category:         "Klinik & Kesehatan",
				Icon:             "🏥",
				SubdomainPattern: "{tenant}.medika.tstech.id",
				BaseDomain:       "medika.tstech.id",
				DemoURL:          "https://demo.medika.tstech.id",
				DocURL:           "https://docs.tstech.id/medika",
				Thumbnail:        "https://images.unsplash.com/photo-1519494026892-80bbd2d6fd0d?w=800&auto=format&fit=crop&q=80",
				Images:           `["https://images.unsplash.com/photo-1519494026892-80bbd2d6fd0d?w=1000&auto=format&fit=crop&q=80"]`,
				Features:         "• **Terintegrasi SATUSEHAT Kemenkes**: Kirim data resume medis RME sesuai regulasi Permenkes No. 24.\n• **Subdomain Khusus Klinik**: Akses aman via `namaklinik.medika.tstech.id`.\n• **Bridging BPJS PCare**: Pendaftaran peserta BPJS & entri tindakan langsung terhubung.\n• **Modul E-Resep & Manajemen Apotek**: Resep dokter langsung masuk ke bagian instalasi farmasi.\n• **Billing Rawat Jalan & Kasir Medis**: Pembayaran tindakan dokter, lab, dan obat terintegrasi.",
				TechStack:        "Next.js, Go, PostgreSQL HIPAA Compliant Encrypted, Redis",
				Overview:         "### Digitalisasi Praktik Dokter & Klinik Pratama Sesuai Standar Kemenkes\n\nMedika EMR mempermudah dokter, perawat, apoteker, dan staf administrasi mengelola klinik tanpa kertas dengan rekam medis terenkripsi standar internasional.",
				IsActive:         true,
				SortOrder:        3,
			},
			Plans: []model.SaaSPlan{
				{
					Name:         "Praktik Dokter Mandiri (Bulanan)",
					Code:         "medika_doctor_monthly",
					Interval:     "monthly",
					PriceMonthly: 249000,
					PriceYearly:  2490000,
					DiscountPct:  0,
					Features:     `["1 Dokter Praktik","RME Rekam Medis SOAP Standar Kemenkes","Integrasi SATUSEHAT","Riwayat Kunjungan Pasien & E-Resep","Cetak Surat Sakit & Rujukan"]`,
					MaxUsers:     3,
					MaxStorageGB: 10,
					IsPopular:    false,
					IsActive:     true,
					SortOrder:    1,
				},
				{
					Name:         "Klinik Pratama (Bulanan)",
					Code:         "medika_klinik_monthly",
					Interval:     "monthly",
					PriceMonthly: 599000,
					PriceYearly:  5990000,
					DiscountPct:  15,
					Features:     `["Hingga 5 Dokter & 10 Nakes","Semua Fitur Praktik Dokter","Antrean Online & Panggil Suara","Modul Kasir & Billing Pasien","Stok Obat & Laporan Farmasi Lengkap","Subdomain klinik dedicated"]`,
					MaxUsers:     15,
					MaxStorageGB: 50,
					IsPopular:    true,
					IsActive:     true,
					SortOrder:    2,
				},
			},
		},
	}

	for _, item := range saasItems {
		prod := item.Product
		if err := db.Create(&prod).Error; err == nil {
			for _, plan := range item.Plans {
				plan.SaaSProductID = prod.ID
				db.Create(&plan)
			}
		}
	}
	log.Println("🚀 Seeded default SaaS Products & Subscription Plans")
}

func seedSampleSubscription(db *gorm.DB) {
	var subCount int64
	db.Model(&model.SaaSSubscription{}).Count(&subCount)
	if subCount > 0 {
		return
	}

	var scholaProd model.SaaSProduct
	var scholaPlan model.SaaSPlan
	if err := db.Where("slug = ?", "schola").First(&scholaProd).Error; err == nil {
		_ = db.Where("saa_s_product_id = ?", scholaProd.ID).First(&scholaPlan)
		now := time.Now()
		demoSub := model.SaaSSubscription{
			SubscriptionNumber: "SUB-DEMO-0001",
			UserID:             2, // Admin TsTech
			SaaSProductID:      scholaProd.ID,
			SaaSPlanID:         scholaPlan.ID,
			TenantName:         "SMK Negeri 1 Surabaya",
			SubdomainSlug:      "smkn1",
			FullSubdomain:      "smkn1.schola.tstech.id",
			Status:             "active",
			BillingCycle:       "yearly",
			PriceAmount:        3000000,
			StartDate:          now.Add(-30 * 24 * time.Hour),
			EndDate:            now.Add(335 * 24 * time.Hour),
			AutoRenew:          true,
		}
		db.Create(&demoSub)
		log.Println("🏫 Seeded sample active subscription for smkn1.schola.tstech.id")
	}
}

func syncDefaultPlanModules(db *gorm.DB) {
	// Schola Starter
	db.Model(&model.SaaSPlan{}).Where("code LIKE ? AND (feature_modules IS NULL OR feature_modules = '')", "%starter%").Updates(map[string]interface{}{
		"feature_modules": `["billing","student_obligations","cash_ledger","infaq","savings","activities","public_website","wa_gateway"]`,
		"allowed_units":   `["sdit"]`,
	})
	// Schola Pro
	db.Model(&model.SaaSPlan{}).Where("code LIKE ? AND (feature_modules IS NULL OR feature_modules = '')", "%pro%").Updates(map[string]interface{}{
		"feature_modules": `["billing","student_obligations","cash_ledger","infaq","savings","activities","public_website","wa_gateway","midtrans","ppdb","rfid_attendance","elearning","bk"]`,
		"allowed_units":   `["sdit","mts","ma"]`,
	})
	// Schola Enterprise
	db.Model(&model.SaaSPlan{}).Where("code LIKE ? AND (feature_modules IS NULL OR feature_modules = '')", "%enterprise%").Updates(map[string]interface{}{
		"feature_modules": `["billing","student_obligations","cash_ledger","infaq","savings","activities","public_website","wa_gateway","midtrans","ppdb","rfid_attendance","elearning","bk","payroll","rkas","assets","external_debts"]`,
		"allowed_units":   `["sdit","mts","ma","tk","umum"]`,
	})
}


