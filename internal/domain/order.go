package domain

import (
	"time"

	"github.com/google/uuid"
)

type OrderStatus string

const (
	OrderPending           OrderStatus = "PENDING"
	OrderPaid              OrderStatus = "PAID"
	OrderFailed            OrderStatus = "FAILED"
)

type Order struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Amount    float64
	IdempotencyKey uuid.UUID
	Status    OrderStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}
