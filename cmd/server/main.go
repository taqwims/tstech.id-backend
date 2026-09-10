package main

import (
	"context"
	"log"

	"github.com/tstech/backend/internal/config"
	"github.com/tstech/backend/internal/database"
	"github.com/tstech/backend/internal/handler"
	"github.com/tstech/backend/internal/repository"
	"github.com/tstech/backend/internal/router"
	"github.com/tstech/backend/internal/service"
	"github.com/labstack/echo/v4"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Connect to database
	db := database.Connect(cfg)

	// Run migrations & seeds
	database.Migrate(db, cfg)

	// Initialize repositories
	consultationRepo := repository.NewConsultationRepo(db)
	orderRepo := repository.NewOrderRepo(db)
	portfolioRepo := repository.NewPortfolioRepo(db)
	testimonialRepo := repository.NewTestimonialRepo(db)
	contactRepo := repository.NewContactRepo(db)
	userRepo := repository.NewUserRepo(db)
	projectRepo := repository.NewProjectRepo(db)
	milestoneRepo := repository.NewMilestoneRepo(db)
	commentRepo := repository.NewCommentRepo(db)
	fileRepo := repository.NewFileRepo(db)
	quotationRepo := repository.NewQuotationRepo(db)
	siteContentRepo := repository.NewSiteContentRepo(db)
	articleRepo := repository.NewArticleRepo(db)
	categoryRepo := repository.NewCategoryRepo(db)
	serviceRepo := repository.NewServiceRepo(db)
	productRepo := repository.NewProductRepo(db)
	saasRepo := repository.NewSaaSRepository(db)
	auditRepo := repository.NewAuditRepo(db)

	// Initialize services
	emailSvc := service.NewEmailService(cfg)
	authSvc := service.NewAuthService(userRepo, cfg)
	storageSvc := service.NewStorageService(cfg)
	consultationSvc := service.NewConsultationService(consultationRepo)
	orderSvc := service.NewOrderService(orderRepo)
	paymentSvc := service.NewPaymentService(db, cfg, orderRepo, userRepo, emailSvc)
	contentSvc := service.NewContentService(siteContentRepo)
	articleSvc := service.NewArticleService(articleRepo)
	serviceSvc := service.NewServiceService(serviceRepo)
	productSvc := service.NewProductService(productRepo)
	saasSvc := service.NewSaaSService(db, cfg, saasRepo, userRepo, paymentSvc)
	auditSvc := service.NewAuditService(auditRepo)
	geminiBlogSvc := service.NewGeminiBlogService(db, articleSvc, auditSvc, cfg)
	projectSvc := service.NewProjectService(
		db,
		projectRepo,
		milestoneRepo,
		commentRepo,
		fileRepo,
		quotationRepo,
		orderRepo,
	)

	// Start background AI blog scheduler worker
	go geminiBlogSvc.StartScheduler(context.Background())

	// Initialize handlers
	handlers := &router.Handlers{
		Consultation: handler.NewConsultationHandler(consultationSvc, emailSvc),
		Order:        handler.NewOrderHandler(orderSvc, paymentSvc, emailSvc, userRepo),
		Payment:      handler.NewPaymentHandler(paymentSvc, storageSvc, auditSvc),
		Portfolio:    handler.NewPortfolioHandler(portfolioRepo),
		Testimonial:  handler.NewTestimonialHandler(testimonialRepo),
		Contact:      handler.NewContactHandler(contactRepo, emailSvc),
		Auth:         handler.NewAuthHandler(authSvc, auditSvc),
		Admin: handler.NewAdminHandler(
			db,
			projectSvc,
			contentSvc,
			storageSvc,
			orderRepo,
			userRepo,
			portfolioRepo,
			testimonialRepo,
			consultationRepo,
			contactRepo,
			auditSvc,
			geminiBlogSvc,
		),
		Client:   handler.NewClientHandler(projectSvc, userRepo, storageSvc, paymentSvc),
		Content:  handler.NewContentHandler(contentSvc),
		Article:  handler.NewArticleHandler(articleSvc),
		Category: handler.NewCategoryHandler(categoryRepo),
		Service:  handler.NewServiceHandler(serviceSvc),
		Product:  handler.NewProductHandler(productSvc),
		SaaS:     handler.NewSaaSHandler(saasSvc, saasRepo, storageSvc),
		AuthSvc:  authSvc,
		Storage:  storageSvc,
	}

	// Setup Echo server
	e := echo.New()
	e.HideBanner = true

	// Setup routes
	router.Setup(e, handlers)

	// Start server
	port := ":" + cfg.APIPort
	log.Printf("🚀 TsTech API server starting on %s (env: %s)", port, cfg.APIEnv)
	if err := e.Start(port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
