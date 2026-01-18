package domain

import (
	"time"

	"github.com/google/uuid"
)

type PaymentStatus string

const (
	PaymentInitiated   PaymentStatus = "INIT"
	PaymentProcessing  PaymentStatus = "PROCESSING"
	PaymentSucceeded PaymentStatus = "SUCCEEDED"
	PaymentFailed      PaymentStatus = "FAILED"
)

type Payment struct {
	ID        uuid.UUID
	OrderID   uuid.UUID
	Amount    float64
	Status    PaymentStatus
	FastPayTxn uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}