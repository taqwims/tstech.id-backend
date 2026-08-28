package handler

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
	"github.com/tstech/backend/internal/model"
	"github.com/tstech/backend/internal/service"
)

type ProductHandler struct {
	svc service.ProductService
}

func NewProductHandler(svc service.ProductService) *ProductHandler {
	return &ProductHandler{svc: svc}
}

// Public: GET /api/products
func (h *ProductHandler) List(c echo.Context) error {
	category := c.QueryParam("category")
	featured := c.QueryParam("featured") == "true"

	list, err := h.svc.ListActive(category, featured)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "Gagal mengambil daftar produk",
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    list,
	})
}

// Public: GET /api/products/featured
func (h *ProductHandler) Featured(c echo.Context) error {
	list, err := h.svc.ListActive("", true)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "Gagal mengambil daftar produk unggulan",
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    list,
	})
}

// Public: GET /api/products/:slug
func (h *ProductHandler) GetBySlug(c echo.Context) error {
	slug := c.Param("slug")
	prod, err := h.svc.GetBySlug(slug)
	if err != nil || prod == nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"success": false,
			"message": "Produk tidak ditemukan",
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    prod,
	})
}

// Admin: GET /api/admin/products
func (h *ProductHandler) AdminList(c echo.Context) error {
	category := c.QueryParam("category")
	search := c.QueryParam("search")

	list, err := h.svc.ListAdmin(category, search)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"success": false,
			"message": "Gagal mengambil daftar produk admin",
		})
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    list,
	})
}

// Admin: GET /api/admin/products/:id
func (h *ProductHandler) AdminGetByID(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "ID tidak valid",
		})
	}

	prod, err := h.svc.GetByID(uint(id))
	if err != nil || prod == nil {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"success": false,
			"message": "Produk tidak ditemukan",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"data":    prod,
	})
}

// Admin: POST /api/admin/products
func (h *ProductHandler) AdminCreate(c echo.Context) error {
	var input model.Product
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
		"message": "Produk berhasil dibuat",
		"data":    input,
	})
}

// Admin: PUT /api/admin/products/:id
func (h *ProductHandler) AdminUpdate(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "ID tidak valid",
		})
	}

	var input model.Product
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
		"message": "Produk berhasil diperbarui",
		"data":    updated,
	})
}

// Admin: DELETE /api/admin/products/:id
func (h *ProductHandler) AdminDelete(c echo.Context) error {
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
			"message": "Gagal menghapus produk",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Produk berhasil dihapus",
	})
}

// Admin: PUT /api/admin/products/:id/toggle-status
func (h *ProductHandler) AdminToggleStatus(c echo.Context) error {
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
		"message": "Status produk berhasil diperbarui",
		"data":    updated,
	})
}

// Admin: PUT /api/admin/products/:id/toggle-featured
func (h *ProductHandler) AdminToggleFeatured(c echo.Context) error {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": "ID tidak valid",
		})
	}

	updated, err := h.svc.ToggleFeatured(uint(id))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"success": false,
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "Status featured produk berhasil diperbarui",
		"data":    updated,
	})
}
