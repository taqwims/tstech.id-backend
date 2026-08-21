package handler

import (
	"net/http"

	"github.com/tstech/backend/internal/service"
	"github.com/tstech/backend/pkg/response"
	"github.com/labstack/echo/v4"
)

type ContentHandler struct {
	contentSvc *service.ContentService
}

func NewContentHandler(contentSvc *service.ContentService) *ContentHandler {
	return &ContentHandler{contentSvc: contentSvc}
}

// GET /api/content
func (h *ContentHandler) GetAll(c echo.Context) error {
	contents, err := h.contentSvc.GetAll()
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mengambil konten")
	}
	return response.Success(c, contents)
}

// GET /api/content/:group
func (h *ContentHandler) GetByGroup(c echo.Context) error {
	group := c.Param("group")
	contents, err := h.contentSvc.GetByGroup(group)
	if err != nil {
		return response.Error(c, http.StatusInternalServerError, "Gagal mengambil konten grup")
	}
	return response.Success(c, contents)
}
