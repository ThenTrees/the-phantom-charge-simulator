package service

import (
	"context"
	"database/sql"
	"errors"
	"math/rand/v2"
	"the-phantom-charge/internal/domain"
	"the-phantom-charge/internal/infrastructure/payment"
	"the-phantom-charge/internal/repo"
	"time"

	"github.com/google/uuid"
)


type OrderService interface {
	Checkout(ctx context.Context, orderId uuid.UUID) (string, error)
	CreateOrder(ctx context.Context) (*domain.Order, error)
}

type orderService struct {
	db           *sql.DB
	orderRepo    repo.OrderRepo
	paymentRepo  repo.PaymentRepo
	paymentGtw   payment.PaymentGateway
}

func NewOrderService(
	db *sql.DB,
	orderRepo repo.OrderRepo,
	paymentRepo repo.PaymentRepo,
	paymentGtw payment.PaymentGateway,
) OrderService {
	return &orderService{
		db:          db,
		orderRepo:   orderRepo,
		paymentRepo: paymentRepo,
		paymentGtw:  paymentGtw,
	}
}

func (s *orderService) Checkout(ctx context.Context, orderId uuid.UUID) (string, error) {
	order, err := s.orderRepo.FindById(ctx, orderId)
	if err != nil {
		return "", err
	}

	if order == nil  {
		return "", errors.New("order not found")
	}

	if order.Status != domain.OrderPending {
		return "", errors.New("order is not in pending state")
	}

	isPaid, err := s.paymentGtw.Charge(ctx, int64(order.Amount), order.IdempotencyKey)
	if err != nil {
		return "", err
	}

	if !isPaid {
		return "", errors.New("payment failed")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}

	defer tx.Rollback()
	order.Status = domain.OrderPaid

	// update order status
	err = s.orderRepo.UpdateOrderStatus(ctx, tx, order)
	if err != nil {
		defer tx.Rollback()
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return "", nil
}

func (os *orderService) CreateOrder(ctx context.Context) (*domain.Order, error) {
	order := &domain.Order{
		ID:           uuid.New(),
		Amount:       rand.Float64() * 10000,
		UserID:       uuid.New(),
		IdempotencyKey: uuid.New(),
		Status:       domain.OrderPending,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	tx, err := os.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	// save order
	err = os.orderRepo.CreateOrder(ctx, tx, order)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return order, nil
}