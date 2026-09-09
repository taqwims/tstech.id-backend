package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/tstech/backend/internal/config"
	"github.com/tstech/backend/internal/model"
	"gorm.io/gorm"
)

type GeminiBlogService interface {
	GetSettings() (*model.AIBlogSetting, error)
	UpdateSettings(setting *model.AIBlogSetting) (*model.AIBlogSetting, error)
	GenerateArticle(customTopic string) (*model.Article, error)
	StartScheduler(ctx context.Context)
}

type geminiBlogService struct {
	db         *gorm.DB
	articleSvc *ArticleService
	auditSvc   AuditService
	cfg        *config.Config
}

func NewGeminiBlogService(
	db *gorm.DB,
	articleSvc *ArticleService,
	auditSvc AuditService,
	cfg *config.Config,
) GeminiBlogService {
	return &geminiBlogService{
		db:         db,
		articleSvc: articleSvc,
		auditSvc:   auditSvc,
		cfg:        cfg,
	}
}

func (s *geminiBlogService) GetSettings() (*model.AIBlogSetting, error) {
	var setting model.AIBlogSetting
	err := s.db.First(&setting).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			setting = model.AIBlogSetting{
				IsEnabled:      false,
				GeminiModel:    "gemini-3-flash-preview",
				IntervalHours:  24,
				TargetCategory: "Teknologi",
				DefaultStatus:  "published",
				AutoSEO:        true,
				LastRunStatus:  "idle",
				TotalGenerated: 0,
			}
			s.db.Create(&setting)
			return &setting, nil
		}
		return nil, err
	}

	// Auto upgrade legacy or deprecated models (1.5, 2.0, 2.5) if found in settings
	if strings.Contains(setting.GeminiModel, "1.5") || strings.Contains(setting.GeminiModel, "2.0") || strings.Contains(setting.GeminiModel, "2.5") {
		setting.GeminiModel = "gemini-3-flash-preview"
		_ = s.db.Save(&setting)
	}

	return &setting, nil
}

func (s *geminiBlogService) UpdateSettings(input *model.AIBlogSetting) (*model.AIBlogSetting, error) {
	setting, err := s.GetSettings()
	if err != nil {
		return nil, err
	}

	setting.IsEnabled = input.IsEnabled
	if input.GeminiAPIKey != "" {
		setting.GeminiAPIKey = input.GeminiAPIKey
	}
	if input.GeminiModel != "" {
		setting.GeminiModel = input.GeminiModel
	}
	if input.IntervalHours > 0 {
		setting.IntervalHours = input.IntervalHours
	}
	if input.TargetCategory != "" {
		setting.TargetCategory = input.TargetCategory
	}
	if input.DefaultStatus != "" {
		setting.DefaultStatus = input.DefaultStatus
	}
	setting.AutoSEO = input.AutoSEO
	setting.PromptTemplate = input.PromptTemplate

	// If enabled and NextRunAt is nil or past, schedule next run
	if setting.IsEnabled {
		if setting.NextRunAt == nil || time.Now().After(*setting.NextRunAt) {
			next := time.Now().Add(time.Duration(setting.IntervalHours) * time.Hour)
			setting.NextRunAt = &next
		}
	}

	if err := s.db.Save(setting).Error; err != nil {
		return nil, err
	}

	s.auditSvc.LogSystem(
		"AI_BLOG_SETTINGS_UPDATE",
		"ai_blog",
		fmt.Sprintf("%d", setting.ID),
		fmt.Sprintf("Update konfigurasi AI Blog Generator (Aktif: %v, Interval: %d jam, Model: %s)", setting.IsEnabled, setting.IntervalHours, setting.GeminiModel),
		setting,
	)

	return setting, nil
}

type geminiGeneratedContent struct {
	Title           string `json:"title"`
	Slug            string `json:"slug"`
	Excerpt         string `json:"excerpt"`
	Category        string `json:"category"`
	Tags            string `json:"tags"`
	Content         string `json:"content"`
	MetaTitle       string `json:"meta_title"`
	MetaDescription string `json:"meta_description"`
	MetaKeywords    string `json:"meta_keywords"`
}

var fallbackTopics = []string{
	"Arsitektur Microservices Modern Berbasis Golang & Next.js untuk Perusahaan",
	"Strategi Mengamankan RESTful API dan Autentikasi JWT dari Serangan Siber",
	"Panduan Implementasi Clean Architecture pada Proyek Web Skala Besar",
	"Optimasi Database PostgreSQL: Indexing, Connection Pooling, dan Query Tuning",
	"Membangun Aplikasi Multi-Tenant SaaS dengan Keamanan Data Terisolasi",
	"Otomasi Alur Bisnis dengan AI dan Integrasi Webhook Payment Gateway",
	"Tren UI/UX Design System 2026: Mengapa Aksesibilitas dan Kecepatan adalah Kunci",
	"Perbandingan Arsitektur Serverless vs VPS Cloud untuk Software House",
}

func (s *geminiBlogService) getCoverImageForCategory(cat string) string {
	catLower := strings.ToLower(cat)
	switch {
	case strings.Contains(catLower, "web"):
		return "https://images.unsplash.com/photo-1460925895917-afdab827c52f?w=1200&auto=format&fit=crop&q=80"
	case strings.Contains(catLower, "mobile"):
		return "https://images.unsplash.com/photo-1512941937669-90a1b58e7e9c?w=1200&auto=format&fit=crop&q=80"
	case strings.Contains(catLower, "seo"):
		return "https://images.unsplash.com/photo-1504868584819-f8e8b4b6d7e3?w=1200&auto=format&fit=crop&q=80"
	case strings.Contains(catLower, "sistem") || strings.Contains(catLower, "erp"):
		return "https://images.unsplash.com/photo-1454165804606-c3d57bc86b40?w=1200&auto=format&fit=crop&q=80"
	case strings.Contains(catLower, "ui") || strings.Contains(catLower, "ux"):
		return "https://images.unsplash.com/photo-1507238691740-187a5b1d37b8?w=1200&auto=format&fit=crop&q=80"
	default:
		return "https://images.unsplash.com/photo-1551288049-bebda4e38f71?w=1200&auto=format&fit=crop&q=80"
	}
}

func (s *geminiBlogService) GenerateArticle(customTopic string) (*model.Article, error) {
	setting, err := s.GetSettings()
	if err != nil {
		return nil, fmt.Errorf("gagal memuat pengaturan AI: %w", err)
	}

	apiKey := strings.TrimSpace(setting.GeminiAPIKey)
	if apiKey == "" {
		apiKey = strings.TrimSpace(os.Getenv("GEMINI_API_KEY"))
	}
	if apiKey == "" {
		errMsg := "API Key Google Gemini belum dikonfigurasi. Harap isi di pengaturan AI Blog atau GEMINI_API_KEY .env"
		setting.LastRunStatus = "error"
		setting.LastRunMessage = errMsg
		now := time.Now()
		setting.LastRunAt = &now
		s.db.Save(setting)
		return nil, errors.New(errMsg)
	}

	modelName := strings.Trim(strings.TrimSpace(setting.GeminiModel), `"'`)
	if envModel := strings.Trim(strings.TrimSpace(os.Getenv("GEMINI_MODEL")), `"'`); envModel != "" {
		modelName = envModel
	}
	if modelName == "" || strings.Contains(modelName, "1.5") || strings.Contains(modelName, "2.0") || strings.Contains(modelName, "2.5") {
		modelName = "gemini-3-flash-preview"
	}

	topic := strings.TrimSpace(customTopic)
	if topic == "" {
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		topic = fallbackTopics[r.Intn(len(fallbackTopics))]
	}

	category := setting.TargetCategory
	if category == "" || category == "All" {
		category = "Teknologi"
	}

	systemInstruction := `Anda adalah Senior Technical Writer dan Lead Software Architect di TsTech (software house profesional di Indonesia).
Tugas Anda adalah menulis artikel blog mendalam, orisinal, bernilai teknis tinggi, dan SEO-friendly dalam Bahasa Indonesia.
Artikel harus memiliki struktur:
- Pendahuluan dengan latar belakang masalah nyata dalam industri software/teknologi
- Poin-poin pembahasan mendalam menggunakan heading markdown (##, ###)
- Contoh kode praktis atau tabel perbandingan jika relevan
- Tips implementasi dan best practices
- Kesimpulan dan CTA profesional ke jasa software house TsTech.

Format output HARUS HANYA berupa objek JSON valid tanpa komentar, tanpa markup teks di luar JSON.
Struktur JSON wajib:
{
  "title": "Judul artikel yang menarik dan memuat keyword utama (maksimal 70 karakter)",
  "slug": "slug-url-ramah-seo",
  "excerpt": "Ringkasan padat 2-3 kalimat untuk kartu artikel dan meta preview",
  "category": "` + category + `",
  "tags": "tag1, tag2, tag3, tag4",
  "content": "Isi lengkap artikel dalam format Markdown rapi",
  "meta_title": "Title tag SEO (maksimal 60 karakter)",
  "meta_description": "Meta description SEO padat (140-160 karakter)",
  "meta_keywords": "keyword1, keyword2, keyword3"
}`

	userPrompt := fmt.Sprintf("Topik artikel: \"%s\". Kategori: \"%s\".", topic, category)
	if setting.PromptTemplate != "" {
		userPrompt += "\nPetunjuk tambahan: " + setting.PromptTemplate
	}

	requestBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": systemInstruction + "\n\n" + userPrompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature":     0.7,
			"maxOutputTokens": 4096,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("gagal serialize request Gemini: %w", err)
	}

	// Models to attempt in priority order (handles temporary 503 overload on any single model)
	modelsToTry := []string{modelName}
	if modelName == "gemini-3-flash-preview" {
		modelsToTry = append(modelsToTry, "gemini-3.6-flash")
	} else if modelName == "gemini-3.6-flash" {
		modelsToTry = append(modelsToTry, "gemini-3-flash-preview")
	} else {
		modelsToTry = append(modelsToTry, "gemini-3-flash-preview", "gemini-3.6-flash")
	}

	client := &http.Client{Timeout: 90 * time.Second}
	var lastErr error
	var respBody []byte

	for _, tryModel := range modelsToTry {
		apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", tryModel, apiKey)
		resp, err := client.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			lastErr = fmt.Errorf("request ke Gemini (%s) gagal: %w", tryModel, err)
			continue
		}
		b, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("gagal membaca response Gemini (%s): %w", tryModel, err)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("Gemini API Error (%s, status %d): %s", tryModel, resp.StatusCode, string(b))
			continue
		}
		respBody = b
		lastErr = nil
		break
	}

	if lastErr != nil || len(respBody) == 0 {
		setting.LastRunStatus = "error"
		setting.LastRunMessage = lastErr.Error()
		now := time.Now()
		setting.LastRunAt = &now
		s.db.Save(setting)
		return nil, lastErr
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(respBody, &geminiResp); err != nil {
		return nil, fmt.Errorf("gagal parse response Gemini: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("Gemini tidak mengembalikan teks konten")
	}

	var rawText string
	for _, part := range geminiResp.Candidates[0].Content.Parts {
		if strings.Contains(part.Text, "{") && strings.Contains(part.Text, "title") {
			rawText = strings.TrimSpace(part.Text)
			break
		}
		if rawText == "" && part.Text != "" {
			rawText = strings.TrimSpace(part.Text)
		}
	}

	// Clean code fences if Gemini returned ```json ... ```
	if strings.HasPrefix(rawText, "```json") {
		rawText = strings.TrimPrefix(rawText, "```json")
	} else if strings.HasPrefix(rawText, "```") {
		rawText = strings.TrimPrefix(rawText, "```")
	}
	rawText = strings.TrimSuffix(rawText, "```")
	rawText = strings.TrimSpace(rawText)

	var generated geminiGeneratedContent
	if err := json.Unmarshal([]byte(rawText), &generated); err != nil {
		// Fallback: try finding first '{' and last '}'
		start := strings.Index(rawText, "{")
		end := strings.LastIndex(rawText, "}")
		if start >= 0 && end > start {
			sub := rawText[start : end+1]
			if err2 := json.Unmarshal([]byte(sub), &generated); err2 != nil {
				return nil, fmt.Errorf("gagal parse JSON dari output AI: %w (raw: %s)", err, rawText)
			}
		} else {
			return nil, fmt.Errorf("output AI bukan JSON valid: %w (raw: %s)", err, rawText)
		}
	}

	coverImage := s.getCoverImageForCategory(generated.Category)

	status := setting.DefaultStatus
	if status == "" {
		status = "published"
	}

	articleInput := CreateArticleInput{
		Title:           generated.Title,
		Slug:            generated.Slug,
		Excerpt:         generated.Excerpt,
		Content:         generated.Content,
		CoverImage:      coverImage,
		Category:        generated.Category,
		Tags:            generated.Tags,
		AuthorName:      "TsTech AI Editorial",
		AuthorAvatar:    "https://images.unsplash.com/photo-1618005182384-a83a8bd57fbe?w=150&auto=format&fit=crop&q=80",
		AuthorRole:      "Automated Tech Insights",
		Status:          status,
		IsFeatured:      false,
		IsTrending:      true,
		MetaTitle:       generated.MetaTitle,
		MetaDescription: generated.MetaDescription,
		MetaKeywords:    generated.MetaKeywords,
	}

	article, err := s.articleSvc.Create(articleInput)
	if err != nil {
		setting.LastRunStatus = "error"
		setting.LastRunMessage = fmt.Sprintf("Gagal menyimpan artikel: %v", err)
		now := time.Now()
		setting.LastRunAt = &now
		s.db.Save(setting)
		return nil, fmt.Errorf("gagal menyimpan artikel AI ke database: %w", err)
	}

	// Update setting status
	now := time.Now()
	next := now.Add(time.Duration(setting.IntervalHours) * time.Hour)
	setting.LastRunAt = &now
	setting.NextRunAt = &next
	setting.LastRunStatus = "success"
	setting.LastRunMessage = fmt.Sprintf("Berhasil generate artikel: \"%s\"", article.Title)
	setting.TotalGenerated++
	s.db.Save(setting)

	// Audit Log
	s.auditSvc.LogSystem(
		"AI_BLOG_GENERATE",
		"article",
		fmt.Sprintf("%d", article.ID),
		fmt.Sprintf("Gemini AI berhasil membuat artikel baru: \"%s\" (%s)", article.Title, article.Slug),
		map[string]interface{}{
			"article_id": article.ID,
			"title":      article.Title,
			"category":   article.Category,
			"model":      modelName,
			"status":     status,
		},
	)

	return article, nil
}

func (s *geminiBlogService) StartScheduler(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	log.Println("🤖 Gemini AI Blog Scheduler worker started (checks every 5m)")

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Stopping Gemini AI Blog Scheduler")
			return
		case <-ticker.C:
			s.checkAndRunScheduled()
		}
	}
}

func (s *geminiBlogService) checkAndRunScheduled() {
	setting, err := s.GetSettings()
	if err != nil || !setting.IsEnabled {
		return
	}

	now := time.Now()
	if setting.NextRunAt == nil || now.After(*setting.NextRunAt) {
		log.Printf("🤖 Running scheduled Gemini AI blog generation...")
		_, err := s.GenerateArticle("")
		if err != nil {
			log.Printf("⚠️ Scheduled Gemini AI generation failed: %v", err)
		} else {
			log.Printf("✅ Scheduled Gemini AI article generation completed successfully")
		}
	}
}
