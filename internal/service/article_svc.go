package service

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/kotban/backend/internal/model"
	"github.com/kotban/backend/internal/repository"
)

type ArticleService struct {
	repo repository.ArticleRepository
}

func NewArticleService(repo repository.ArticleRepository) *ArticleService {
	return &ArticleService{repo: repo}
}

type CreateArticleInput struct {
	Title           string `json:"title"`
	Slug            string `json:"slug"`
	Excerpt         string `json:"excerpt"`
	Content         string `json:"content"`
	CoverImage      string `json:"cover_image"`
	Category        string `json:"category"`
	Tags            string `json:"tags"`
	AuthorName      string `json:"author_name"`
	AuthorAvatar    string `json:"author_avatar"`
	AuthorRole      string `json:"author_role"`
	Status          string `json:"status"`
	IsFeatured      bool   `json:"is_featured"`
	IsTrending      bool   `json:"is_trending"`
	MetaTitle       string `json:"meta_title"`
	MetaDescription string `json:"meta_description"`
	MetaKeywords    string `json:"meta_keywords"`
	CanonicalURL    string `json:"canonical_url"`
}

type UpdateArticleInput struct {
	Title           string `json:"title"`
	Slug            string `json:"slug"`
	Excerpt         string `json:"excerpt"`
	Content         string `json:"content"`
	CoverImage      string `json:"cover_image"`
	Category        string `json:"category"`
	Tags            string `json:"tags"`
	AuthorName      string `json:"author_name"`
	AuthorAvatar    string `json:"author_avatar"`
	AuthorRole      string `json:"author_role"`
	Status          string `json:"status"`
	IsFeatured      bool   `json:"is_featured"`
	IsTrending      bool   `json:"is_trending"`
	MetaTitle       string `json:"meta_title"`
	MetaDescription string `json:"meta_description"`
	MetaKeywords    string `json:"meta_keywords"`
	CanonicalURL    string `json:"canonical_url"`
}

func (s *ArticleService) GenerateSlug(title string) string {
	slug := strings.ToLower(title)
	// Replace non-alphanumeric characters with hyphen
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	slug = reg.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = fmt.Sprintf("artikel-%d", time.Now().Unix())
	}
	return slug
}

func (s *ArticleService) CalculateReadingTime(content string) int {
	// Strip basic HTML/markdown tags
	reg := regexp.MustCompile(`<[^>]*>`)
	plainText := reg.ReplaceAllString(content, " ")
	words := strings.Fields(plainText)
	wordCount := len(words)
	// Average reading speed: 200 words per minute
	minutes := int(math.Ceil(float64(wordCount) / 200.0))
	if minutes < 1 {
		minutes = 1
	}
	return minutes
}

func (s *ArticleService) Create(input CreateArticleInput) (*model.Article, error) {
	if strings.TrimSpace(input.Title) == "" {
		return nil, errors.New("judul artikel wajib diisi")
	}
	if strings.TrimSpace(input.Content) == "" {
		return nil, errors.New("konten artikel wajib diisi")
	}

	slug := strings.TrimSpace(input.Slug)
	if slug == "" {
		slug = s.GenerateSlug(input.Title)
	} else {
		slug = s.GenerateSlug(slug)
	}

	// Check if slug exists
	existing, _ := s.repo.FindBySlug(slug)
	if existing != nil {
		slug = fmt.Sprintf("%s-%d", slug, time.Now().Unix()%10000)
	}

	readingTime := s.CalculateReadingTime(input.Content)

	category := strings.TrimSpace(input.Category)
	if category == "" {
		category = "Teknologi"
	}

	authorName := strings.TrimSpace(input.AuthorName)
	if authorName == "" {
		authorName = "Tim Redaksi Kotban"
	}

	authorRole := strings.TrimSpace(input.AuthorRole)
	if authorRole == "" {
		authorRole = "Tech Writer & Engineer"
	}

	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = "published"
	}

	var publishedAt *time.Time
	if status == "published" {
		now := time.Now()
		publishedAt = &now
	}

	excerpt := strings.TrimSpace(input.Excerpt)
	if excerpt == "" {
		// Take first 160 characters from plain text content
		reg := regexp.MustCompile(`<[^>]*>`)
		clean := reg.ReplaceAllString(input.Content, " ")
		clean = strings.TrimSpace(clean)
		if len(clean) > 160 {
			excerpt = clean[:157] + "..."
		} else {
			excerpt = clean
		}
	}

	metaTitle := strings.TrimSpace(input.MetaTitle)
	if metaTitle == "" {
		metaTitle = input.Title
	}

	metaDesc := strings.TrimSpace(input.MetaDescription)
	if metaDesc == "" {
		metaDesc = excerpt
	}

	article := &model.Article{
		Title:           input.Title,
		Slug:            slug,
		Excerpt:         excerpt,
		Content:         input.Content,
		CoverImage:      input.CoverImage,
		Category:        category,
		Tags:            input.Tags,
		AuthorName:      authorName,
		AuthorAvatar:    input.AuthorAvatar,
		AuthorRole:      authorRole,
		Status:          status,
		IsFeatured:      input.IsFeatured,
		IsTrending:      input.IsTrending,
		ReadingTime:     readingTime,
		MetaTitle:       metaTitle,
		MetaDescription: metaDesc,
		MetaKeywords:    input.MetaKeywords,
		CanonicalURL:    input.CanonicalURL,
		PublishedAt:     publishedAt,
	}

	if err := s.repo.Create(article); err != nil {
		return nil, err
	}

	return article, nil
}

func (s *ArticleService) Update(id uint, input UpdateArticleInput) (*model.Article, error) {
	article, err := s.repo.FindByID(id)
	if err != nil {
		return nil, errors.New("artikel tidak ditemukan")
	}

	if strings.TrimSpace(input.Title) != "" {
		article.Title = input.Title
	}
	if strings.TrimSpace(input.Content) != "" {
		article.Content = input.Content
		article.ReadingTime = s.CalculateReadingTime(input.Content)
	}
	if strings.TrimSpace(input.Slug) != "" {
		newSlug := s.GenerateSlug(input.Slug)
		if newSlug != article.Slug {
			existing, _ := s.repo.FindBySlug(newSlug)
			if existing != nil && existing.ID != article.ID {
				return nil, errors.New("slug sudah digunakan oleh artikel lain")
			}
			article.Slug = newSlug
		}
	}

	article.Excerpt = input.Excerpt
	article.CoverImage = input.CoverImage
	if strings.TrimSpace(input.Category) != "" {
		article.Category = input.Category
	}
	article.Tags = input.Tags
	if strings.TrimSpace(input.AuthorName) != "" {
		article.AuthorName = input.AuthorName
	}
	if strings.TrimSpace(input.AuthorAvatar) != "" {
		article.AuthorAvatar = input.AuthorAvatar
	}
	if strings.TrimSpace(input.AuthorRole) != "" {
		article.AuthorRole = input.AuthorRole
	}

	if input.Status != "" && input.Status != article.Status {
		article.Status = input.Status
		if input.Status == "published" && article.PublishedAt == nil {
			now := time.Now()
			article.PublishedAt = &now
		}
	}

	article.IsFeatured = input.IsFeatured
	article.IsTrending = input.IsTrending
	article.MetaTitle = input.MetaTitle
	article.MetaDescription = input.MetaDescription
	article.MetaKeywords = input.MetaKeywords
	article.CanonicalURL = input.CanonicalURL

	if err := s.repo.Update(article); err != nil {
		return nil, err
	}

	return article, nil
}

func (s *ArticleService) Delete(id uint) error {
	return s.repo.Delete(id)
}

func (s *ArticleService) GetByID(id uint) (*model.Article, error) {
	return s.repo.FindByID(id)
}

func (s *ArticleService) GetBySlug(slug string) (*model.Article, []model.Article, error) {
	article, err := s.repo.FindBySlug(slug)
	if err != nil {
		return nil, nil, errors.New("artikel tidak ditemukan")
	}

	// Increment view count asynchronously/safely
	_ = s.repo.IncrementViews(article.ID)

	// Fetch related articles
	related, _ := s.repo.GetRelated(article.Category, article.ID, 3)

	return article, related, nil
}

func (s *ArticleService) List(page, perPage int, category, tag, search, status string) ([]model.Article, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 50 {
		perPage = 10
	}
	return s.repo.List(page, perPage, category, tag, search, status)
}

func (s *ArticleService) GetFeatured(limit int) ([]model.Article, error) {
	if limit < 1 {
		limit = 5
	}
	return s.repo.GetFeatured(limit)
}

func (s *ArticleService) GetTrending(limit int) ([]model.Article, error) {
	if limit < 1 {
		limit = 5
	}
	return s.repo.GetTrending(limit)
}

func (s *ArticleService) GetCategories() ([]model.ArticleCategoryCount, error) {
	return s.repo.GetCategoriesWithCount()
}

func (s *ArticleService) GetAllPublished() ([]model.Article, error) {
	return s.repo.GetAllPublished()
}

func (s *ArticleService) ToggleStatus(id uint) (*model.Article, error) {
	article, err := s.repo.FindByID(id)
	if err != nil {
		return nil, errors.New("artikel tidak ditemukan")
	}

	if article.Status == "published" {
		article.Status = "draft"
	} else {
		article.Status = "published"
		if article.PublishedAt == nil {
			now := time.Now()
			article.PublishedAt = &now
		}
	}

	if err := s.repo.Update(article); err != nil {
		return nil, err
	}

	return article, nil
}
