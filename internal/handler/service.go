package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/tstech/backend/internal/model"
	"github.com/tstech/backend/internal/service"
)

type ServiceHandler struct {
	svc service.ServiceService
}

func NewServiceHandler(svc service.ServiceService) *ServiceHandler {
	return &ServiceHandler{svc: svc}
}

// Public: GET /api/services
func (h *ServiceHandler) List(c echo.Context) error {
	list, err := h.svc.ListActive()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "Gagal mengambil daftar layanan",
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    list,
	})
}

// Public: GET /api/services/:slug
func (h *ServiceHandler) GetBySlug(c echo.Context) error {
	slug := c.Param("slug")
	svc, err := h.svc.GetBySlug(slug)
	if err != nil || svc == nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"success": false,
			"message": "Layanan tidak ditemukan",
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    svc,
	})
}

// Admin: GET /api/admin/services
func (h *ServiceHandler) AdminList(c echo.Context) error {
	list, err := h.svc.ListAdmin()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "Gagal mengambil daftar layanan admin",
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    list,
	})
}

// Admin: GET /api/admin/services/:id
func (h *ServiceHandler) AdminGetByID(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "ID tidak valid",
		})
	}

	svc, err := h.svc.GetByID(uint(id))
	if err != nil || svc == nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"success": false,
			"message": "Layanan tidak ditemukan",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    svc,
	})
}

// Admin: POST /api/admin/services
func (h *ServiceHandler) AdminCreate(c echo.Context) error {
	var input model.Service
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "Format data tidak valid: " + err.Error(),
		})
	}

	if err := h.svc.Create(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusCreated, map[string]interface{}{
		"success": true,
		"message": "Layanan berhasil dibuat",
		"data":    input,
	})
}

// Admin: PUT /api/admin/services/:id
func (h *ServiceHandler) AdminUpdate(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "ID tidak valid",
		})
	}

	var input model.Service
	if err := c.Bind(&input); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "Format data tidak valid: " + err.Error(),
		})
	}

	updated, err := h.svc.Update(uint(id), &input)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Layanan berhasil diperbarui",
		"data":    updated,
	})
}

// Admin: DELETE /api/admin/services/:id
func (h *ServiceHandler) AdminDelete(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "ID tidak valid",
		})
	}

	if err := h.svc.Delete(uint(id)); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "Gagal menghapus layanan",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Layanan berhasil dihapus",
	})
}

// Admin: PUT /api/admin/services/:id/toggle-status
func (h *ServiceHandler) AdminToggleStatus(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "ID tidak valid",
		})
	}

	updated, err := h.svc.ToggleStatus(uint(id))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Status layanan berhasil diperbarui",
		"data":    updated,
	})
}
