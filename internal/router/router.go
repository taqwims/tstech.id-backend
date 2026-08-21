package router

import (
	"github.com/tstech/backend/internal/handler"
	"github.com/tstech/backend/internal/middleware"
	"github.com/tstech/backend/internal/service"
	"github.com/labstack/echo/v4"
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
	AuthSvc      *service.AuthService
}

func Setup(e *echo.Echo, h *Handlers) {
	// Global middleware
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(middleware.CORS())

	// Static uploads directory
	e.Static("/uploads", "./uploads")

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
	auth.POST("/refresh", h.Auth.Refresh)
	auth.GET("/me", h.Auth.Me, middleware.JWTAuth(h.AuthSvc))

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
	client.GET("/projects", h.Client.ListProjects)
	client.GET("/projects/:id", h.Client.GetProject)
	client.POST("/projects/:id/comments", h.Client.AddComment)
	client.POST("/projects/:id/files", h.Client.UploadFile)
	client.PUT("/projects/:id/quotation/respond", h.Client.RespondQuotation)
	client.POST("/projects/:id/pay-balance", h.Client.CreateBalancePayment)

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

	// Orders & Conversion
	admin.GET("/orders", h.Admin.ListOrders)
	admin.PUT("/orders/:id/status", h.Admin.UpdateOrderStatus)
	admin.POST("/orders/:id/convert-to-project", h.Admin.ConvertOrderToProject)

	// Portfolios Management
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

	// User Management
	admin.GET("/users", h.Admin.ListUsers)
	admin.PUT("/users/:id/toggle-status", h.Admin.ToggleUserStatus)

	// Payment Gateway & Transactions Management
	admin.GET("/payment-settings", h.Payment.GetAdminSettings)
	admin.PUT("/payment-settings", h.Payment.UpdateAdminSettings)
	admin.GET("/payments", h.Payment.ListAdminPayments)
	admin.PUT("/payments/:id/verify", h.Payment.VerifyAdminPayment)

	// File Upload
	admin.POST("/upload", h.Admin.UploadFile)
}
