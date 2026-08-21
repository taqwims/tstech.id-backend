package service_test

import (
	"testing"

	"github.com/kotban/backend/internal/config"
	"github.com/kotban/backend/internal/model"
	"github.com/kotban/backend/internal/repository"
	"github.com/kotban/backend/internal/service"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) (*gorm.DB, *service.PaymentService, *repository.OrderRepo, *repository.UserRepo) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	assert.NoError(t, err)

	err = db.AutoMigrate(
		&model.User{},
		&model.Order{},
		&model.Payment{},
		&model.Project{},
		&model.Milestone{},
		&model.ProjectComment{},
		&model.PaymentSetting{},
	)
	assert.NoError(t, err)

	cfg := &config.Config{
		AppURL: "http://localhost:3000",
	}

	orderRepo := repository.NewOrderRepo(db)
	userRepo := repository.NewUserRepo(db)
	emailSvc := service.NewEmailService(cfg)
	paymentSvc := service.NewPaymentService(db, cfg, orderRepo, userRepo, emailSvc)

	return db, paymentSvc, orderRepo, userRepo
}

func TestPakasirWebhook(t *testing.T) {
	db, paymentSvc, _, userRepo := setupTestDB(t)

	// Create test client user
	clientUser := &model.User{
		Email: "client@test.com",
		Name:  "Test Client",
		Role:  "client",
	}
	assert.NoError(t, userRepo.Create(clientUser))

	// Create test order
	order := &model.Order{
		OrderNumber:     "ORD-202608-0001",
		CustomerName:    "Test Client",
		CustomerEmail:   "client@test.com",
		CustomerWA:      "08123456789",
		ProjectName:     "Web E-Commerce",
		ProjectDesc:     "Toko online modern",
		TotalAmount:     10000000,
		DPAmount:        5000000,
		Status:          "pending_dp",
		PackageType:     "professional",
		ServiceCategory: "Website E-commerce",
	}
	assert.NoError(t, db.Create(order).Error)

	// Create payment
	payment := &model.Payment{
		InvoiceNumber:  "INV/202608/0001",
		OrderID:        &order.ID,
		Amount:         5000000,
		PaymentType:    "dp",
		PaymentMethod:  "pakasir",
		GatewayTransID: "KTB-1",
		Status:         "pending",
	}
	assert.NoError(t, db.Create(payment).Error)

	// Simulate Pakasir webhook
	payload := map[string]interface{}{
		"order_id":       "KTB-1",
		"amount":         5000000,
		"status":         "paid",
		"project":        "kotban",
		"payment_method": "qris",
	}

	err := paymentSvc.HandlePakasirNotification(payload)
	assert.NoError(t, err)

	// Verify Payment status is success
	var updatedPayment model.Payment
	db.First(&updatedPayment, payment.ID)
	assert.Equal(t, "success", updatedPayment.Status)
	assert.NotNil(t, updatedPayment.PaidAt)

	// Verify Order status is dp_paid
	var updatedOrder model.Order
	db.First(&updatedOrder, order.ID)
	assert.Equal(t, "dp_paid", updatedOrder.Status)

	// Verify Project was automatically created
	var project model.Project
	err = db.Where("order_id = ?", order.ID).First(&project).Error
	assert.NoError(t, err)
	assert.Equal(t, clientUser.ID, project.ClientUserID)
	assert.Equal(t, int64(5000000), project.PaidAmount)
	assert.Equal(t, "briefing", project.Status)

	// Verify 6 milestones created
	var milestones []model.Milestone
	db.Where("project_id = ?", project.ID).Find(&milestones)
	assert.Equal(t, 6, len(milestones))
}

func TestMayarWebhook(t *testing.T) {
	db, paymentSvc, _, userRepo := setupTestDB(t)

	// Create test client user
	clientUser := &model.User{
		Email: "mayar@test.com",
		Name:  "Mayar User",
		Role:  "client",
	}
	assert.NoError(t, userRepo.Create(clientUser))

	// Create test order
	order := &model.Order{
		OrderNumber:   "ORD-202608-0002",
		CustomerName:  "Mayar User",
		CustomerEmail: "mayar@test.com",
		CustomerWA:    "08123456789",
		ProjectName:   "Company Profile",
		TotalAmount:   6000000,
		DPAmount:      3000000,
		Status:        "pending_dp",
	}
	assert.NoError(t, db.Create(order).Error)

	// Create payment
	payment := &model.Payment{
		InvoiceNumber:  "INV/202608/0002",
		OrderID:        &order.ID,
		Amount:         3000000,
		PaymentType:    "dp",
		PaymentMethod:  "mayar",
		GatewayTransID: "MAYAR-TX-999",
		Status:         "pending",
	}
	assert.NoError(t, db.Create(payment).Error)

	// Simulate Mayar webhook
	payload := map[string]interface{}{
		"event": "payment.received",
		"data": map[string]interface{}{
			"id":            "MAYAR-TX-999",
			"status":        "paid",
			"amount":        3000000,
			"transactionId": "TX-12345",
		},
	}

	err := paymentSvc.HandleMayarNotification(payload)
	assert.NoError(t, err)

	// Verify payment success
	var updatedPayment model.Payment
	db.First(&updatedPayment, payment.ID)
	assert.Equal(t, "success", updatedPayment.Status)

	// Verify project created
	var project model.Project
	err = db.Where("order_id = ?", order.ID).First(&project).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(3000000), project.PaidAmount)
}

func TestIpaymuWebhook(t *testing.T) {
	db, paymentSvc, _, userRepo := setupTestDB(t)

	// Create test client user
	clientUser := &model.User{
		Email: "ipaymu@test.com",
		Name:  "Ipaymu User",
		Role:  "client",
	}
	assert.NoError(t, userRepo.Create(clientUser))

	// Create test order
	order := &model.Order{
		OrderNumber:   "ORD-202608-0003",
		CustomerName:  "Ipaymu User",
		CustomerEmail: "ipaymu@test.com",
		CustomerWA:    "08123456789",
		ProjectName:   "Mobile App",
		TotalAmount:   15000000,
		DPAmount:      7500000,
		Status:        "pending_dp",
	}
	assert.NoError(t, db.Create(order).Error)

	// Create payment
	payment := &model.Payment{
		InvoiceNumber:  "INV/202608/0003",
		OrderID:        &order.ID,
		Amount:         7500000,
		PaymentType:    "dp",
		PaymentMethod:  "ipaymu",
		IpaymuTransID:  "1234567",
		GatewayTransID: "1234567",
		Status:         "pending",
	}
	assert.NoError(t, db.Create(payment).Error)

	// Simulate iPaymu webhook
	payload := map[string]interface{}{
		"trx_id":       "1234567",
		"status":       "berhasil",
		"status_code":  "1",
		"reference_id": "INV/202608/0003",
		"amount":       7500000,
	}

	err := paymentSvc.HandleIpaymuNotification(payload)
	assert.NoError(t, err)

	// Verify payment success
	var updatedPayment model.Payment
	db.First(&updatedPayment, payment.ID)
	assert.Equal(t, "success", updatedPayment.Status)

	// Verify project created
	var project model.Project
	err = db.Where("order_id = ?", order.ID).First(&project).Error
	assert.NoError(t, err)
	assert.Equal(t, int64(7500000), project.PaidAmount)
}
