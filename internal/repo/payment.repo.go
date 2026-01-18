package repo

import (
	"context"
	"database/sql"
	"the-phantom-charge/internal/domain"
	"time"

	"github.com/google/uuid"
)
type PaymentRepo interface {
	// tx *sql.Tx -> kiểm soát transaction
	CreatePayment(ctx context.Context, tx *sql.Tx, payment *domain.Payment) error
  // id uuid.UUID -> tìm kiếm theo id
	FindById(ctx context.Context, id uuid.UUID) (*domain.Payment, error)
	// update order status when charge success
	UpdatePaymentStatus(ctx context.Context, tx *sql.Tx, orderId uuid.UUID, status domain.PaymentStatus, fastPayTxn uuid.UUID) error
	FindProcessingBefore(
		ctx context.Context,
		before time.Time,
		limit int,
	) ([]domain.Payment, error)
}

type paymentRepo struct {
	db *sql.DB
}

func NewPaymentRepo(db *sql.DB) PaymentRepo {
	return &paymentRepo{db: db}
}

func (r *paymentRepo) CreatePayment(ctx context.Context, tx *sql.Tx, payment *domain.Payment) error {
	query := `INSERT INTO payments (id, order_id, amount, fastpay_txn_id, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := tx.ExecContext(
		ctx, query, payment.ID, payment.OrderID, payment.Amount, payment.FastPayTxn, payment.Status, payment.CreatedAt, payment.UpdatedAt,
	)

	if err != nil {
		return err
	}
	return nil
}

func (r *paymentRepo) FindById(ctx context.Context, id uuid.UUID) (*domain.Payment, error)  {
	query := `SELECT * FROM payments WHERE id = $1`
	row := r.db.QueryRowContext(ctx, query, id)
	var p domain.Payment
	err := row.Scan(
		&p.ID,
		&p.OrderID,
		&p.Amount,
		&p.FastPayTxn,
		&p.Status,
		&p.CreatedAt,
		&p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *paymentRepo) UpdatePaymentStatus(ctx context.Context, tx *sql.Tx, orderId uuid.UUID, status domain.PaymentStatus, fastPayTxn uuid.UUID) error {
	query := `
		UPDATE payments
		SET status = $2,
		    fastpay_txn_id = COALESCE($3, fastpay_txn_id),
		    updated_at = now()
		WHERE id = $1
	`
	_, err := tx.ExecContext(
		ctx,
		query,
		orderId,
		status,
		fastPayTxn,
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *paymentRepo) FindProcessingBefore(ctx context.Context, before time.Time, limit int) ([]domain.Payment, error) {
	query := `
		SELECT * FROM payments
		WHERE status = $1
		AND created_at < $2
		LIMIT $3
	`
	rows, err := r.db.QueryContext(ctx, query, domain.PaymentProcessing, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var payments []domain.Payment
	for rows.Next() {
		var p domain.Payment
		err := rows.Scan(
			&p.ID,
			&p.OrderID,
			&p.Amount,
			&p.FastPayTxn,
			&p.Status,
			&p.CreatedAt,
			&p.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		payments = append(payments, p)
	}
	return payments, nil
}