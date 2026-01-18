package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"the-phantom-charge/internal/domain"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/joho/godotenv/autoload"
)

// Service represents a service that interacts with a database.
type Service interface {
	// Health returns a map of health status information.
	// The keys and values in the map are service-specific.
	Health() map[string]string

	// Close terminates the database connection.
	// It returns an error if the connection cannot be closed.
	Close() error
}

type service struct {
	db *sql.DB
}

var (
	database   = os.Getenv("BLUEPRINT_DB_DATABASE")
	password   = os.Getenv("BLUEPRINT_DB_PASSWORD")
	username   = os.Getenv("BLUEPRINT_DB_USERNAME")
	port       = os.Getenv("BLUEPRINT_DB_PORT")
	host       = os.Getenv("BLUEPRINT_DB_HOST")
	schema     = os.Getenv("BLUEPRINT_DB_SCHEMA")
	dbInstance *service
)

func NewPostgres() *sql.DB {
	connStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=%s",
		os.Getenv("BLUEPRINT_DB_USERNAME"),
		os.Getenv("BLUEPRINT_DB_PASSWORD"),
		os.Getenv("BLUEPRINT_DB_HOST"),
		os.Getenv("BLUEPRINT_DB_PORT"),
		os.Getenv("BLUEPRINT_DB_DATABASE"),
		os.Getenv("BLUEPRINT_DB_SCHEMA"),
	)

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatal(err)
	}

	return db
}

func New() Service {
	// Reuse Connection
	if dbInstance != nil {
		return dbInstance
	}
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=%s", username, password, host, port, database, schema)
	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatal(err)
	}
	dbInstance = &service{
		db: db,
	}
	return dbInstance
}

// Health checks the health of the database connection by pinging the database.
// It returns a map with keys indicating various health statistics.
func (s *service) Health() map[string]string {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	stats := make(map[string]string)

	// Ping the database
	err := s.db.PingContext(ctx)
	if err != nil {
		stats["status"] = "down"
		stats["error"] = fmt.Sprintf("db down: %v", err)
		log.Fatalf("db down: %v", err) // Log the error and terminate the program
		return stats
	}

	// Database is up, add more statistics
	stats["status"] = "up"
	stats["message"] = "It's healthy"

	// Get database stats (like open connections, in use, idle, etc.)
	dbStats := s.db.Stats()
	stats["open_connections"] = strconv.Itoa(dbStats.OpenConnections)
	stats["in_use"] = strconv.Itoa(dbStats.InUse)
	stats["idle"] = strconv.Itoa(dbStats.Idle)
	stats["wait_count"] = strconv.FormatInt(dbStats.WaitCount, 10)
	stats["wait_duration"] = dbStats.WaitDuration.String()
	stats["max_idle_closed"] = strconv.FormatInt(dbStats.MaxIdleClosed, 10)
	stats["max_lifetime_closed"] = strconv.FormatInt(dbStats.MaxLifetimeClosed, 10)

	// Evaluate stats to provide a health message
	if dbStats.OpenConnections > 40 { // Assuming 50 is the max for this example
		stats["message"] = "The database is experiencing heavy load."
	}

	if dbStats.WaitCount > 1000 {
		stats["message"] = "The database has a high number of wait events, indicating potential bottlenecks."
	}

	if dbStats.MaxIdleClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many idle connections are being closed, consider revising the connection pool settings."
	}

	if dbStats.MaxLifetimeClosed > int64(dbStats.OpenConnections)/2 {
		stats["message"] = "Many connections are being closed due to max lifetime, consider increasing max lifetime or revising the connection usage pattern."
	}

	return stats
}

// Close closes the database connection.
// It logs a message indicating the disconnection from the specific database.
// If the connection is successfully closed, it returns nil.
// If an error occurs while closing the connection, it returns the error.
func (s *service) Close() error {
	log.Printf("Disconnected from database: %s", database)
	return s.db.Close()
}


func (r *service) CreatePayment(ctx context.Context, tx *sql.Tx, p *domain.Payment) error {
	query := `INSERT INTO payments (id, order_id, amount, fastpay_txn_id, status, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := tx.ExecContext(
		ctx, query, p.ID, p.OrderID, p.Amount, p.FastPayTxn, p.Status, p.CreatedAt, p.UpdatedAt,
	)

	if err != nil {
		return err
	}
	return nil
}

func (r *service) FindById(ctx context.Context, id uuid.UUID) (*domain.Payment, error)  {
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

func (r *service) UpdatePaymentStatus(ctx context.Context, tx *sql.Tx, orderId uuid.UUID, status domain.PaymentStatus, fastPayTxn uuid.UUID) error {
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