package handler

import (
	"net/http"

	"github.com/tstech/backend/internal/repository"
	"github.com/tstech/backend/pkg/response"
	"github.com/labstack/echo/v4"
)

type PortfolioHandler struct {
	repo *repository.PortfolioRepo
}

func NewPortfolioHandler(repo *repository.PortfolioRepo) *PortfolioHandler {
	return &PortfolioHandler{repo: repo}
}

// List returns all portfolios, optionally filtered by category
func (h *PortfolioHandler) List(c echo.Context) error {
	category := c.QueryParam("category")

	portfolios, err := h.repo.FindAll(category)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mengambil data portofolio")
	}

	return response.Success(c, portfolios)
}

// Featured returns featured portfolios for the landing page
func (h *PortfolioHandler) Featured(c echo.Context) error {
	portfolios, err := h.repo.FindFeatured(6)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mengambil data portofolio")
	}

	return response.Success(c, portfolios)
}

// GetBySlug returns a single portfolio by slug
func (h *PortfolioHandler) GetBySlug(c echo.Context) error {
	slug := c.Param("slug")

	portfolio, err := h.repo.FindBySlug(slug)
	if err != nil {
		return response.Error(c, http.StatusNotFound, "Portofolio tidak ditemukan")
	}

	return response.Success(c, portfolio)
}
