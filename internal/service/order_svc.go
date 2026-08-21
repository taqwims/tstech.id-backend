package service

import (
	"fmt"
	"time"

	"github.com/tstech/backend/internal/model"
	"github.com/tstech/backend/internal/repository"
)

type OrderService struct {
	orderRepo *repository.OrderRepo
}

func NewOrderService(orderRepo *repository.OrderRepo) *OrderService {
	return &OrderService{orderRepo: orderRepo}
}

func (s *OrderService) CreateOrder(order *model.Order) error {
	// Generate order number: KTB-YYYYMMDD-XXXX
	order.OrderNumber = s.generateOrderNumber()
	order.Status = "pending"

	return s.orderRepo.Create(order)
}

func (s *OrderService) GetByOrderNumber(orderNumber string) (*model.Order, error) {
	return s.orderRepo.FindByOrderNumber(orderNumber)
}

func (s *OrderService) GetByID(id uint) (*model.Order, error) {
	return s.orderRepo.FindByID(id)
}

func (s *OrderService) UpdateStatus(id uint, status string) error {
	return s.orderRepo.UpdateStatus(id, status)
}

func (s *OrderService) generateOrderNumber() string {
	now := time.Now()
	return fmt.Sprintf("KTB-%s-%04d", now.Format("20060102"), now.UnixNano()%10000)
}
