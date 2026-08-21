package handler

import (
	"net/http"
	"strconv"

	"github.com/tstech/backend/internal/service"
	"github.com/labstack/echo/v4"
)

type ArticleHandler struct {
	svc *service.ArticleService
}

func NewArticleHandler(svc *service.ArticleService) *ArticleHandler {
	return &ArticleHandler{svc: svc}
}

// ==================== PUBLIC ENDPOINTS ====================

func (h *ArticleHandler) List(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	if perPage < 1 {
		perPage = 9
	}

	category := c.QueryParam("category")
	tag := c.QueryParam("tag")
	search := c.QueryParam("search")

	articles, total, err := h.svc.List(page, perPage, category, tag, search, "published")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "Gagal memuat artikel: " + err.Error(),
		})
	}

	totalPages := int((total + int64(perPage) - 1) / int64(perPage))

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    articles,
		"meta": map[string]interface{}{
			"page":        page,
			"per_page":    perPage,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

func (h *ArticleHandler) GetBySlug(c echo.Context) error {
	slug := c.Param("slug")
	if slug == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "Slug artikel tidak valid",
		})
	}

	article, related, err := h.svc.GetBySlug(slug)
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data": map[string]interface{}{
			"article": article,
			"related": related,
		},
	})
}

func (h *ArticleHandler) Featured(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 {
		limit = 3
	}

	articles, err := h.svc.GetFeatured(limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "Gagal memuat artikel featured: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    articles,
	})
}

func (h *ArticleHandler) Trending(c echo.Context) error {
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 {
		limit = 5
	}

	articles, err := h.svc.GetTrending(limit)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "Gagal memuat artikel trending: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    articles,
	})
}

func (h *ArticleHandler) Categories(c echo.Context) error {
	categories, err := h.svc.GetCategories()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "Gagal memuat kategori: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    categories,
	})
}

func (h *ArticleHandler) Sitemap(c echo.Context) error {
	articles, err := h.svc.GetAllPublished()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "Gagal memuat data sitemap: " + err.Error(),
		})
	}

	type SitemapItem struct {
		Slug        string `json:"slug"`
		Title       string `json:"title"`
		UpdatedAt   string `json:"updated_at"`
		Category    string `json:"category"`
		PublishedAt string `json:"published_at"`
	}

	var items []SitemapItem
	for _, a := range articles {
		pubStr := ""
		if a.PublishedAt != nil {
			pubStr = a.PublishedAt.Format("2006-01-02T15:04:05Z07:00")
		}
		items = append(items, SitemapItem{
			Slug:        a.Slug,
			Title:       a.Title,
			UpdatedAt:   a.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Category:    a.Category,
			PublishedAt: pubStr,
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    items,
	})
}

// ==================== ADMIN ENDPOINTS ====================

func (h *ArticleHandler) AdminList(c echo.Context) error {
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}

	perPage, _ := strconv.Atoi(c.QueryParam("per_page"))
	if perPage < 1 {
		perPage = 20
	}

	category := c.QueryParam("category")
	tag := c.QueryParam("tag")
	search := c.QueryParam("search")
	status := c.QueryParam("status")

	articles, total, err := h.svc.List(page, perPage, category, tag, search, status)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "Gagal memuat artikel: " + err.Error(),
		})
	}

	totalPages := int((total + int64(perPage) - 1) / int64(perPage))

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    articles,
		"meta": map[string]interface{}{
			"page":        page,
			"per_page":    perPage,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

func (h *ArticleHandler) AdminGetByID(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "ID artikel tidak valid",
		})
	}

	article, err := h.svc.GetByID(uint(id))
	if err != nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"success": false,
			"message": "Artikel tidak ditemukan",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    article,
	})
}

func (h *ArticleHandler) AdminCreate(c echo.Context) error {
	var input service.CreateArticleInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "Data permintaan tidak valid: " + err.Error(),
		})
	}

	article, err := h.svc.Create(input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"success": true,
		"message": "Artikel berhasil dibuat",
		"data":    article,
	})
}

func (h *ArticleHandler) AdminUpdate(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "ID artikel tidak valid",
		})
	}

	var input service.UpdateArticleInput
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "Data permintaan tidak valid: " + err.Error(),
		})
	}

	article, err := h.svc.Update(uint(id), input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Artikel berhasil diperbarui",
		"data":    article,
	})
}

func (h *ArticleHandler) AdminDelete(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "ID artikel tidak valid",
		})
	}

	if err := h.svc.Delete(uint(id)); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "Gagal menghapus artikel: " + err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Artikel berhasil dihapus",
	})
}

func (h *ArticleHandler) AdminToggleStatus(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "ID artikel tidak valid",
		})
	}

	article, err := h.svc.ToggleStatus(uint(id))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Status artikel berhasil diubah",
		"data":    article,
	})
}
