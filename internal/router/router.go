package router

import (
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/tstech/backend/internal/handler"
	"github.com/tstech/backend/internal/middleware"
	"github.com/tstech/backend/internal/service"
)

type Handlers struct {
	Consultation *handler.ConsultationHandler
	Order        *handler.OrderHandler
	Payment      *handler.PaymentHandler
	Portfolio    *handler.PortfolioHandler
	Testimonial  *handler.TestimonialHandler
	Contact      *handler.ContactHandler
	Auth         *handler.AuthHandler
	Admin        *handler.AdminHandler
	Client       *handler.ClientHandler
	Content      *handler.ContentHandler
	Article      *handler.ArticleHandler
	Category     *handler.CategoryHandler
	Service      *handler.ServiceHandler
	Product      *handler.ProductHandler
	SaaS         *handler.SaaSHandler
	AuthSvc      *service.AuthService
	Storage      *service.StorageService
}

func Setup(e *echo.Echo, h *Handlers) {
	// Global middleware
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(middleware.CORS())

	// Static uploads directory with caching headers & R2 cloud fallback
	e.Match([]string{"GET", "HEAD"}, "/uploads/*", func(c echo.Context) error {
		param := c.Param("*")
		cleanParam := strings.TrimPrefix(filepath.ToSlash(filepath.Clean(param)), "/")
		if cleanParam == "" || cleanParam == "." || strings.HasPrefix(cleanParam, "..") {
			return echo.ErrNotFound
		}
		if h.Storage != nil {
			rc, contentType, err := h.Storage.GetFile(cleanParam)
			if err == nil {
				defer rc.Close()
				c.Response().Header().Set("Cache-Control", "public, max-age=2592000, immutable")
				if contentType != "" {
					c.Response().Header().Set("Content-Type", contentType)
				}
				if c.Request().Method == "HEAD" {
					return c.NoContent(http.StatusOK)
				}
				_, err = io.Copy(c.Response().Writer, rc)
				return err
			}
		}
		return echo.ErrNotFound
	})

	// API group
	api := e.Group("/api")

	// Health check
	api.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{"status": "ok", "service": "tstech-api"})
	})

	// Public CMS Content
	api.GET("/content", h.Content.GetAll)
	api.GET("/content/:group", h.Content.GetByGroup)

	// Auth Endpoints
	auth := api.Group("/auth")
	auth.POST("/register", h.Auth.Register)
	auth.POST("/login", h.Auth.Login)
	auth.POST("/firebase", h.Auth.FirebaseLogin)
	auth.POST("/refresh", h.Auth.Refresh)
	auth.GET("/me", h.Auth.Me, middleware.JWTAuth(h.AuthSvc))
	auth.POST("/sso/verify", h.Auth.VerifySSO)
	auth.GET("/sso/verify", h.Auth.VerifySSO)

	// Consultations
	api.POST("/consultations", h.Consultation.Create)

	// Orders
	api.POST("/orders", h.Order.Create)
	api.GET("/orders/:orderNumber", h.Order.GetByOrderNumber)

	// Payments & Verification
	api.GET("/payments/methods", h.Payment.GetMethods)
	api.POST("/payments/create", h.Payment.Create)
	api.POST("/payments/notify", h.Payment.Notify)
	api.POST("/payments/notify/pakasir", h.Payment.NotifyPakasir)
	api.POST("/payments/notify/mayar", h.Payment.NotifyMayar)
	api.POST("/payments/notify/ipaymu", h.Payment.Notify)
	api.POST("/payments/:id/proof", h.Payment.UploadProof)
	api.POST("/payments/:id/sync", h.Payment.SyncStatus)
	api.GET("/payments/status/:identifier", h.Payment.GetStatusAndSync)
	api.GET("/verify", h.Payment.VerifyDocument)
	api.GET("/verify/:number", h.Payment.VerifyDocument)

	// Portfolios
	api.GET("/portfolios", h.Portfolio.List)
	api.GET("/portfolios/featured", h.Portfolio.Featured)
	api.GET("/portfolios/:slug", h.Portfolio.GetBySlug)

	// Testimonials
	api.GET("/testimonials", h.Testimonial.List)

	// Articles & Blog
	api.GET("/articles", h.Article.List)
	api.GET("/articles/featured", h.Article.Featured)
	api.GET("/articles/trending", h.Article.Trending)
	api.GET("/articles/categories", h.Article.Categories)
	api.GET("/articles/sitemap", h.Article.Sitemap)
	api.GET("/articles/:slug", h.Article.GetBySlug)

	// Categories (Blog & Portfolio)
	api.GET("/categories", h.Category.List)

	// Services
	api.GET("/services", h.Service.List)
	api.GET("/services/:slug", h.Service.GetBySlug)

	// Products (Public)
	api.GET("/products", h.Product.List)
	api.GET("/products/featured", h.Product.Featured)
	api.GET("/products/:slug", h.Product.GetBySlug)

	// SaaS Catalog, Subdomain Checker & License Verification (Public)
	api.GET("/saas/products", h.SaaS.GetProducts)
	api.GET("/saas/products/:slug", h.SaaS.GetProductDetail)
	api.GET("/saas/check-subdomain", h.SaaS.CheckSubdomain)
	api.GET("/saas/license/verify", h.SaaS.VerifyLicense)
	api.GET("/saas/verify", h.SaaS.VerifyLicense)
	api.POST("/webhooks/mayar", h.SaaS.HandleMayarWebhook)

	// Site Content (Public)
	api.GET("/content", h.Admin.ListContent)

	// Contacts
	contacts := api.Group("/contacts")
	contacts.Use(middleware.RateLimiter())
	contacts.POST("", h.Contact.Create)

	// ==================== CLIENT PROTECTED ROUTES ====================
	client := api.Group("/client")
	client.Use(middleware.JWTAuth(h.AuthSvc))
	client.Use(middleware.RequireRole("client", "admin"))

	client.GET("/profile", h.Client.GetProfile)
	client.PUT("/profile", h.Client.UpdateProfile)
	client.GET("/orders", h.Client.ListOrders)
	client.GET("/projects", h.Client.ListProjects)
	client.GET("/projects/:id", h.Client.GetProject)
	client.POST("/projects/:id/comments", h.Client.AddComment)
	client.POST("/projects/:id/files", h.Client.UploadFile)
	client.PUT("/projects/:id/quotation/respond", h.Client.RespondQuotation)
	client.POST("/projects/:id/pay-balance", h.Client.CreateBalancePayment)

	// Client SaaS Subscriptions & Invoices
	client.POST("/saas/subscribe", h.SaaS.Subscribe)
	client.GET("/saas/subscriptions", h.SaaS.GetUserSubscriptions)
	client.GET("/saas/subscriptions/:id", h.SaaS.GetSubscriptionDetail)
	client.POST("/saas/subscriptions/:id/sso-token", h.SaaS.GenerateSSOToken)
	client.GET("/invoices", h.SaaS.GetUserInvoices)
	client.GET("/invoices/:id", h.SaaS.GetInvoiceDetail)
	client.POST("/invoices/:id/manual-proof", h.SaaS.SubmitManualProof)

	// ==================== ADMIN PROTECTED ROUTES ====================
	admin := api.Group("/admin")
	admin.Use(middleware.JWTAuth(h.AuthSvc))
	admin.Use(middleware.RequireRole("admin"))

	// Dashboard & Projects
	admin.GET("/dashboard", h.Admin.Dashboard)
	admin.GET("/projects", h.Admin.ListProjects)
	admin.POST("/projects", h.Admin.CreateProject)
	admin.GET("/projects/:id", h.Admin.GetProject)
	admin.PUT("/projects/:id", h.Admin.UpdateProject)
	admin.PUT("/projects/:id/status", h.Admin.UpdateProjectStatus)
	admin.PUT("/projects/:id/progress", h.Admin.UpdateProjectProgress)
	admin.POST("/projects/:id/comments", h.Admin.AddProjectComment)
	admin.POST("/projects/:id/files", h.Admin.UploadProjectFile)
	admin.POST("/projects/:id/payments", h.Admin.RecordProjectPayment)

	// Milestones
	admin.POST("/projects/:id/milestones", h.Admin.CreateMilestone)
	admin.PUT("/milestones/:id", h.Admin.UpdateMilestone)
	admin.PUT("/milestones/:id/complete", h.Admin.CompleteMilestone)
	admin.DELETE("/milestones/:id", h.Admin.DeleteMilestone)

	// Quotation
	admin.POST("/projects/:id/quotation", h.Admin.SaveQuotation)
	admin.PUT("/quotations/:id/send", h.Admin.SendQuotation)
	admin.PUT("/quotations/:id/bypass", h.Admin.ApproveQuotationBypass)

	// Orders & Conversion
	admin.GET("/orders", h.Admin.ListOrders)
	admin.PUT("/orders/:id/status", h.Admin.UpdateOrderStatus)
	admin.POST("/orders/:id/convert-to-project", h.Admin.ConvertOrderToProject)

	// Portfolios Management
	admin.GET("/portfolios/:id", h.Admin.GetPortfolioByID)
	admin.POST("/portfolios", h.Admin.CreatePortfolio)
	admin.PUT("/portfolios/:id", h.Admin.UpdatePortfolio)
	admin.DELETE("/portfolios/:id", h.Admin.DeletePortfolio)

	// Testimonials Management
	admin.GET("/testimonials", h.Admin.ListTestimonials)
	admin.POST("/testimonials", h.Admin.CreateTestimonial)
	admin.PUT("/testimonials/:id", h.Admin.UpdateTestimonial)
	admin.DELETE("/testimonials/:id", h.Admin.DeleteTestimonial)

	// Consultations & Contacts Management
	admin.GET("/consultations", h.Admin.ListConsultations)
	admin.PUT("/consultations/:id/status", h.Admin.UpdateConsultationStatus)
	admin.GET("/contacts", h.Admin.ListContacts)

	// CMS Content Management
	admin.GET("/content", h.Admin.ListContent)
	admin.PUT("/content", h.Admin.UpdateContent)

	// Blog & Articles Management
	admin.GET("/articles", h.Article.AdminList)
	admin.GET("/articles/:id", h.Article.AdminGetByID)
	admin.POST("/articles", h.Article.AdminCreate)
	admin.PUT("/articles/:id", h.Article.AdminUpdate)
	admin.DELETE("/articles/:id", h.Article.AdminDelete)
	admin.PUT("/articles/:id/toggle-status", h.Article.AdminToggleStatus)

	// Category Management (Blog & Portfolio)
	admin.GET("/categories", h.Category.AdminList)
	admin.POST("/categories", h.Category.Create)
	admin.PUT("/categories/:id", h.Category.Update)
	admin.DELETE("/categories/:id", h.Category.Delete)

	// Services Management
	admin.GET("/services", h.Service.AdminList)
	admin.GET("/services/:id", h.Service.AdminGetByID)
	admin.POST("/services", h.Service.AdminCreate)
	admin.PUT("/services/:id", h.Service.AdminUpdate)
	admin.DELETE("/services/:id", h.Service.AdminDelete)
	admin.PUT("/services/:id/toggle-status", h.Service.AdminToggleStatus)

	// Products Management & SaaS Plans
	admin.GET("/products", h.Product.AdminList)
	admin.GET("/products/:id", h.Product.AdminGetByID)
	admin.POST("/products", h.Product.AdminCreate)
	admin.PUT("/products/:id", h.Product.AdminUpdate)
	admin.DELETE("/products/:id", h.Product.AdminDelete)
	admin.PUT("/products/:id/toggle-status", h.Product.AdminToggleStatus)
	admin.PUT("/products/:id/toggle-featured", h.Product.AdminToggleFeatured)
	admin.POST("/products/:id/plans", h.Product.AdminCreatePlan)
	admin.PUT("/plans/:id", h.Product.AdminUpdatePlan)
	admin.DELETE("/plans/:id", h.Product.AdminDeletePlan)

	// User Management
	admin.GET("/users", h.Admin.ListUsers)
	admin.PUT("/users/:id/toggle-status", h.Admin.ToggleUserStatus)

	// Payment Gateway & Transactions Management
	admin.GET("/payment-settings", h.Payment.GetAdminSettings)
	admin.PUT("/payment-settings", h.Payment.UpdateAdminSettings)
	admin.GET("/payments", h.Payment.ListAdminPayments)
	admin.PUT("/payments/:id/verify", h.Payment.VerifyAdminPayment)

	// SaaS Subscriptions & Invoices Management (Admin)
	admin.GET("/saas/stats", h.SaaS.AdminGetStats)
	admin.GET("/saas/products", h.SaaS.AdminGetAllProducts)
	admin.POST("/saas/products", h.SaaS.AdminCreateProduct)
	admin.PUT("/saas/products/:id", h.SaaS.AdminUpdateProduct)
	admin.PUT("/saas/products/:id/toggle-featured", h.SaaS.AdminToggleFeatured)
	admin.DELETE("/saas/products/:id", h.SaaS.AdminDeleteProduct)
	admin.POST("/saas/products/:id/plans", h.SaaS.AdminCreatePlan)
	admin.PUT("/saas/plans/:id", h.SaaS.AdminUpdatePlan)
	admin.DELETE("/saas/plans/:id", h.SaaS.AdminDeletePlan)
	admin.GET("/saas/subscriptions", h.SaaS.AdminGetAllSubscriptions)
	admin.POST("/saas/subscriptions", h.SaaS.AdminCreateSubscription)
	admin.PUT("/saas/subscriptions/:id", h.SaaS.AdminUpdateSubscription)
	admin.DELETE("/saas/subscriptions/:id", h.SaaS.AdminDeleteSubscription)
	admin.GET("/invoices", h.SaaS.AdminGetAllInvoices)
	admin.POST("/invoices/:id/approve", h.SaaS.AdminApproveInvoice)

	// File Upload
	admin.POST("/upload", h.Admin.UploadFile)

	// Audit Logs
	admin.GET("/audit-logs", h.Admin.ListAuditLogs)

	// AI Blog Automation (Gemini)
	admin.GET("/ai-blog/settings", h.Admin.GetAIBlogSettings)
	admin.PUT("/ai-blog/settings", h.Admin.UpdateAIBlogSettings)
	admin.POST("/ai-blog/generate-now", h.Admin.GenerateAIBlogNow)
}
