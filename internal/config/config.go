package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	// Database
	DatabaseURL string
	DBDriver    string
	DBHost      string
	DBPort      string
	DBUser      string
	DBPassword  string
	DBName      string
	DBSSLMode   string

	// API
	APIPort string
	APIEnv  string

	// JWT
	JWTSecret          string
	JWTExpireHours     int
	JWTRefreshExpHours int

	// Admin Seed
	AdminEmail    string
	AdminPassword string
	AdminName     string

	// Pakasir
	PakasirProjectSlug string
	PakasirAPIKey      string
	PakasirQRISOnly    bool

	// Mayar
	MayarAPIKey       string
	MayarIsProduction bool
	MayarWebhookToken string

	// iPaymu
	IpaymuVA           string
	IpaymuAPIKey       string
	IpaymuIsProduction bool
	IpaymuReturnURL    string
	IpaymuCancelURL    string
	IpaymuNotifyURL    string

	// SMTP
	SMTPHost      string
	SMTPPort      int
	SMTPUser      string
	SMTPPassword  string
	SMTPFromName  string
	SMTPFromEmail string

	// Cloudflare R2
	R2AccountID       string
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2BucketName      string
	R2PublicURL       string

	// WhatsApp
	WhatsAppNumber string

	// SaaS & SSO Domains
	SaaSBaseDomain     string
	SaaSSSOProtocol    string
	SaaSDevURLOverride string
	HubAPIURL          string

	// App
	AppURL  string
	AppName string
}

func Load() *Config {
	// Auto-load .env file if running locally
	loadDotEnv()

	smtpPort, _ := strconv.Atoi(getEnv("SMTP_PORT", "587"))
	ipaymuProd, _ := strconv.ParseBool(getEnv("IPAYMU_IS_PRODUCTION", "false"))
	pakasirQRISOnly, _ := strconv.ParseBool(getEnv("PAKASIR_QRIS_ONLY", "false"))
	jwtExpire, _ := strconv.Atoi(getEnv("JWT_EXPIRE_HOURS", "24"))
	jwtRefresh, _ := strconv.Atoi(getEnv("JWT_REFRESH_EXPIRE_HOURS", "168"))

	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", ""),
		DBDriver:    getEnv("DB_DRIVER", "postgres"),
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "5432"),
		DBUser:      getEnv("DB_USER", "tstech"),
		DBPassword:  getEnv("DB_PASSWORD", "tstech_secret_2024"),
		DBName:      getEnv("DB_NAME", "tstech_db"),
		DBSSLMode:   getEnv("DB_SSLMODE", "disable"),

		APIPort: getEnv("PORT", getEnv("API_PORT", "8080")),
		APIEnv:  getEnv("API_ENV", "development"),

		JWTSecret:          getEnv("JWT_SECRET", "tstech-super-secret-key-change-me"),
		JWTExpireHours:     jwtExpire,
		JWTRefreshExpHours: jwtRefresh,

		AdminEmail:    getEnv("ADMIN_EMAIL", "admin@tstech.id"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
		AdminName:     getEnv("ADMIN_NAME", "Admin TsTech"),

		PakasirProjectSlug: getEnv("PAKASIR_PROJECT_SLUG", "tstech"),
		PakasirAPIKey:      getEnv("PAKASIR_API_KEY", ""),
		PakasirQRISOnly:    pakasirQRISOnly,

		MayarAPIKey:       getEnv("MAYAR_API_KEY", ""),
		MayarIsProduction: getEnv("MAYAR_IS_PRODUCTION", "false") == "true",
		MayarWebhookToken: getEnv("MAYAR_WEBHOOK_TOKEN", ""),

		IpaymuVA:           getEnv("IPAYMU_VA", ""),
		IpaymuAPIKey:       getEnv("IPAYMU_API_KEY", ""),
		IpaymuIsProduction: ipaymuProd,
		IpaymuReturnURL:    getEnv("IPAYMU_RETURN_URL", "http://localhost:3000/pemesanan/sukses"),
		IpaymuCancelURL:    getEnv("IPAYMU_CANCEL_URL", "http://localhost:3000/pemesanan/batal"),
		IpaymuNotifyURL:    getEnv("IPAYMU_NOTIFY_URL", "http://localhost:8080/api/payments/notify"),

		SMTPHost:      getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:      smtpPort,
		SMTPUser:      getEnv("SMTP_USER", ""),
		SMTPPassword:  getEnv("SMTP_PASSWORD", ""),
		SMTPFromName:  getEnv("SMTP_FROM_NAME", "TsTech"),
		SMTPFromEmail: getEnv("SMTP_FROM_EMAIL", "noreply@tstech.id"),

		R2AccountID:       getEnv("R2_ACCOUNT_ID", ""),
		R2AccessKeyID:     getEnv("R2_ACCESS_KEY_ID", ""),
		R2SecretAccessKey: getEnv("R2_SECRET_ACCESS_KEY", ""),
		R2BucketName:      getEnv("R2_BUCKET_NAME", "tstech-files"),
		R2PublicURL:       getEnv("R2_PUBLIC_URL", ""),

		WhatsAppNumber: getEnv("WHATSAPP_NUMBER", "628xxxxxxxxxx"),

		SaaSBaseDomain:     getEnv("SAAS_BASE_DOMAIN", "tstech.id"),
		SaaSSSOProtocol:    getEnv("SAAS_SSO_PROTOCOL", "https"),
		SaaSDevURLOverride: getEnv("SAAS_DEV_URL_OVERRIDE", ""),
		HubAPIURL:          getEnv("HUB_API_URL", "http://localhost:8080/api"),

		AppURL:  getEnv("APP_URL", "http://localhost:3000"),
		AppName: getEnv("APP_NAME", "TsTech"),
	}
}

func (c *Config) DSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	return "host=" + c.DBHost +
		" user=" + c.DBUser +
		" password=" + c.DBPassword +
		" dbname=" + c.DBName +
		" port=" + c.DBPort +
		" sslmode=" + c.DBSSLMode +
		" TimeZone=Asia/Jakarta"
}

func loadDotEnv() {
	paths := []string{".env", "../.env", "../../.env"}
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			continue
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				// Clean quotes if present
				value = strings.Trim(value, `"'`)
				os.Setenv(key, value)
			}
		}
		break
	}
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && val != "" {
		return val
	}
	return fallback
}
