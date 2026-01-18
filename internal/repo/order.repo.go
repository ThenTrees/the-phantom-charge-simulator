package repo

import (
	"context"
	"database/sql"
	"the-phantom-charge/internal/domain"
	"time"

	"github.com/google/uuid"
)

type OrderRepo interface {
	FindById(ctx context.Context, id uuid.UUID) (*domain.Order, error)
	UpdateOrderStatus(ctx context.Context, tx *sql.Tx, order *domain.Order) error
	CreateOrder(ctx context.Context, tx *sql.Tx, order *domain.Order) error
	FindStuckOrders(ctx context.Context, olderThan time.Duration) ([]domain.Order, error)
}

type orderRepo struct {
	db *sql.DB
}

func NewOrderRepo(db *sql.DB) OrderRepo {
	return &orderRepo{db: db}
}

func (r *orderRepo) FindById(ctx context.Context, id uuid.UUID) (*domain.Order, error) {
	var order domain.Order
	err := r.db.QueryRowContext(ctx, "SELECT * FROM orders WHERE id = $1", id).Scan(
		&order.ID,
		&order.UserID,
		&order.Amount,
		&order.IdempotencyKey,
		&order.Status,
		&order.CreatedAt,
		&order.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // not found
	}
	if err != nil {
		return nil, err // system error
	}
	return &order, nil
}

func (r *orderRepo) UpdateOrderStatus(ctx context.Context, tx *sql.Tx, order *domain.Order) error {
	_, err := tx.ExecContext(ctx, "UPDATE orders SET status = $1, updated_at = $2 WHERE id = $3", order.Status, order.UpdatedAt, order.ID)
	if err != nil {
		return err
	}
	return nil
}

func (or *orderRepo) CreateOrder(ctx context.Context, tx *sql.Tx, order *domain.Order) error {
	_, err := tx.ExecContext(ctx, "INSERT INTO orders (id, user_id, amount, status, idempotency_key, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)", order.ID, order.UserID, order.Amount, order.Status, order.IdempotencyKey, order.CreatedAt, order.UpdatedAt)
	if err != nil {
		return err
	}
	return nil
}


func (or *orderRepo) FindStuckOrders(ctx context.Context, olderThan time.Duration) ([]domain.Order, error) {
	var orders []domain.Order

	rows, err := or.db.QueryContext(ctx,
		"SELECT * FROM orders WHERE status = 'PENDING' AND updated_at < $1",
		time.Now().Add(-olderThan),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var order domain.Order
		if err := rows.Scan(
			&order.ID,
			&order.UserID,
			&order.Amount,
			&order.IdempotencyKey,
			&order.Status,
			&order.CreatedAt,
			&order.UpdatedAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, nil
}