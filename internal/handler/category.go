package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/tstech/backend/internal/model"
	"github.com/tstech/backend/internal/repository"
	"github.com/tstech/backend/pkg/response"
)

type CategoryHandler struct {
	repo *repository.CategoryRepo
}

func NewCategoryHandler(repo *repository.CategoryRepo) *CategoryHandler {
	return &CategoryHandler{repo: repo}
}

// GET /api/categories?type=blog|portfolio
func (h *CategoryHandler) List(c echo.Context) error {
	catType := c.QueryParam("type")
	categories, err := h.repo.FindAll(catType)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mengambil data kategori")
	}
	return response.Success(c, categories)
}

// GET /api/admin/categories?type=blog|portfolio
func (h *CategoryHandler) AdminList(c echo.Context) error {
	catType := c.QueryParam("type")
	categories, err := h.repo.FindAllAdmin(catType)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mengambil data kategori admin")
	}
	return response.Success(c, categories)
}

// POST /api/admin/categories
func (h *CategoryHandler) Create(c echo.Context) error {
	var cat model.Category
	if err := c.Bind(&cat); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format input tidak valid")
	}

	if cat.Name == "" {
		return response.Error(c, http.StatusBadRequest, "Nama kategori wajib diisi")
	}

	if cat.Type == "" {
		cat.Type = "blog"
	}

	if cat.Slug == "" {
		cat.Slug = strings.ToLower(strings.ReplaceAll(cat.Name, " ", "-"))
		cat.Slug = strings.ReplaceAll(cat.Slug, "&", "dan")
		cat.Slug = strings.ReplaceAll(cat.Slug, "/", "-")
	}

	// Check unique slug within type
	existing, _ := h.repo.FindBySlug(cat.Type, cat.Slug)
	if existing != nil {
		cat.Slug = fmt.Sprintf("%s-%d", cat.Slug, time.Now().Unix())
	}

	cat.IsActive = true
	if err := h.repo.Create(&cat); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal membuat kategori")
	}

	return response.Created(c, cat, "Kategori berhasil ditambahkan")
}

// PUT /api/admin/categories/:id
func (h *CategoryHandler) Update(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "ID kategori tidak valid")
	}

	cat, err := h.repo.FindByID(uint(id))
	if err != nil || cat == nil {
		return response.Error(c, http.StatusNotFound, "Kategori tidak ditemukan")
	}

	var req struct {
		Name        string `json:"name"`
		Slug        string `json:"slug"`
		Type        string `json:"type"`
		Description string `json:"description"`
		Icon        string `json:"icon"`
		SortOrder   int    `json:"sort_order"`
		IsActive    *bool  `json:"is_active"`
	}

	if err := c.Bind(&req); err != nil {
		return response.Error(c, http.StatusBadRequest, "Format input tidak valid")
	}

	if req.Name != "" {
		cat.Name = req.Name
	}
	if req.Slug != "" {
		cat.Slug = req.Slug
	}
	if req.Type != "" {
		cat.Type = req.Type
	}
	cat.Description = req.Description
	cat.Icon = req.Icon
	cat.SortOrder = req.SortOrder
	if req.IsActive != nil {
		cat.IsActive = *req.IsActive
	}

	if err := h.repo.Update(cat); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal memperbarui kategori")
	}

	return response.SuccessWithMessage(c, cat, "Kategori berhasil diperbarui")
}

// DELETE /api/admin/categories/:id
func (h *CategoryHandler) Delete(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return response.Error(c, http.StatusBadRequest, "ID kategori tidak valid")
	}

	if err := h.repo.Delete(uint(id)); err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal menghapus kategori")
	}

	return response.SuccessWithMessage(c, nil, "Kategori berhasil dihapus")
}
