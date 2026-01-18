package payment

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/google/uuid"
)

type PaymentGateway interface {
    Charge(ctx context.Context, amount int64, idempotencyKey uuid.UUID) (bool, error)
		CheckStatus(ctx context.Context, idempotencyKey uuid.UUID) (bool, error)
}

type paymentGateway struct {
	mu           	sync.RWMutex
	chargeSuccess map[string]bool
}

func NewPaymentGateway() PaymentGateway {
	chargeSuccess := make(map[string]bool)
	return &paymentGateway{chargeSuccess: chargeSuccess}
}

func (pg *paymentGateway) Charge(ctx context.Context, amount int64, idempotencyKey uuid.UUID) (bool, error) {
	key := idempotencyKey.String()

	// check Idempotency Key (if charged, return true)
	pg.mu.RLock()
	if paid, exists := pg.chargeSuccess[key]; exists {
		pg.mu.RUnlock()
		return paid, nil
	}
	pg.mu.RUnlock()

	// Tạo số ngẫu nhiên từ 0 đến 99
	chance := rand.IntN(100)

	switch {
	// --- TRƯỜNG HỢP 1: THÀNH CÔNG (70%) ---
	case chance < 70:
		time.Sleep(100 * time.Millisecond)
		pg.mu.Lock()
		pg.chargeSuccess[idempotencyKey.String()] = true
		pg.mu.Unlock()
		return true, nil

	// --- TRƯỜNG HỢP 2: THẺ LỖI (20%) ---
	case chance < 90:
		time.Sleep(100 * time.Millisecond)
		pg.mu.Lock()
		pg.chargeSuccess[idempotencyKey.String()] = false
		pg.mu.Unlock()
		return false, errors.New("Card Declined")

	// --- TRƯỜNG HỢP 3: MẠNG LAG - THE PHANTOM CHARGE (10%) ---
	default:
		// Giả lập mạng bị treo 2 giây
		time.Sleep(2 * time.Second)
		pg.mu.Lock()
		pg.chargeSuccess[idempotencyKey.String()] = true
		pg.mu.Unlock()
		// THẢM HỌA: Bên FastPay đã thực hiện trừ tiền thành công
		fmt.Printf("[FastPay] CHARGED MONEY for Key: %s\n", idempotencyKey)

		// Nhưng Backend của mình lại nhận về lỗi Timeout (hoặc chủ động trả về lỗi)
		return false, errors.New("Connection Timeout")
	}
}

func (pg *paymentGateway) CheckStatus(ctx context.Context, idempotencyKey uuid.UUID) (bool, error) {
	pg.mu.RLock()
	defer pg.mu.RUnlock()

	// Giả lập check status API
	if paid, exists := pg.chargeSuccess[idempotencyKey.String()]; exists {
		return paid, nil
	}
	return false, nil // Chưa thấy giao dịch này
}