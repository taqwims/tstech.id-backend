package response

import (
	"math"
	"net/http"

	"github.com/labstack/echo/v4"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

type Meta struct {
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

func Success(c echo.Context, data interface{}) error {
	return c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
	})
}

func SuccessWithMessage(c echo.Context, data interface{}, message string) error {
	return c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func Created(c echo.Context, data interface{}, message string) error {
	return c.JSON(http.StatusCreated, APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func SuccessWithMeta(c echo.Context, data interface{}, meta *Meta) error {
	return c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

func Paginated(c echo.Context, data interface{}, page, perPage int, total int64) error {
	totalPages := 1
	if perPage > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(perPage)))
	}
	return c.JSON(http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
		Meta: &Meta{
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

func Error(c echo.Context, code int, message string) error {
	return c.JSON(code, APIResponse{
		Success: false,
		Message: message,
	})
}

func ValidationError(c echo.Context, errors interface{}) error {
	return c.JSON(http.StatusUnprocessableEntity, APIResponse{
		Success: false,
		Message: "Validation failed",
		Errors:  errors,
	})
}
