package handler

import (
	"net/http"

	"github.com/kotban/backend/internal/repository"
	"github.com/kotban/backend/pkg/response"
	"github.com/labstack/echo/v4"
)

type TestimonialHandler struct {
	repo *repository.TestimonialRepo
}

func NewTestimonialHandler(repo *repository.TestimonialRepo) *TestimonialHandler {
	return &TestimonialHandler{repo: repo}
}

// List returns all active testimonials
func (h *TestimonialHandler) List(c echo.Context) error {
	testimonials, err := h.repo.FindActive()
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mengambil data testimoni")
	}

	return response.Success(c, testimonials)
}
