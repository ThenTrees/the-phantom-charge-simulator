package worker

import (
	"context"
	"database/sql"
	"log"
	"the-phantom-charge/internal/domain"
	"the-phantom-charge/internal/infrastructure/payment"
	"the-phantom-charge/internal/repo"
	"time"
)

type ReconciliationWorker struct {
	db *sql.DB
	orderRepo repo.OrderRepo
	gateway   payment.PaymentGateway
	interval  time.Duration
}

func NewReconciliationWorker(
	db *sql.DB,
	orderRepo repo.OrderRepo,
	gateway payment.PaymentGateway,
	interval time.Duration,
) *ReconciliationWorker {
	return &ReconciliationWorker{
		db:        db,
		orderRepo: orderRepo,
		gateway:   gateway,
		interval:  interval,
	}
}

func (rw *ReconciliationWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(rw.interval)
	defer ticker.Stop()

	log.Println("Reconciliation worker started")

	for {
		select {
		case <-ctx.Done(): // Worker bị dừng
			return
		case <-ticker.C:   // Đến giờ chạy job
			// Logic xử lý chính ở đây
			if err := rw.process(ctx); err != nil {
				log.Printf("Reconciliation failed: %v", err)
			}
		}
	}
}

// process thực hiện logic đối soát
func (rw *ReconciliationWorker) process(ctx context.Context) error {
	// 1. Tìm các đơn "PENDING" đã tạo quá 1 phút trước (nghĩa là bị kẹt)
	stuckOrders, err := rw.orderRepo.FindStuckOrders(ctx, 1*time.Minute)
	if err != nil {
		return err
	}

	if len(stuckOrders) == 0 {
		return nil // Không có đơn nào bị kẹt
	}

	log.Printf("Found %d stuck orders. Fixing...", len(stuckOrders))

	// 2. Duyệt từng đơn và fix
	for _, order := range stuckOrders {
		// Gọi sang MockGateway để hỏi: Đơn này Status thực tế là gì?
		isPaid, err := rw.gateway.CheckStatus(ctx, order.IdempotencyKey)
		if err != nil {
			log.Printf("Failed to check status for order %s: %v", order.ID, err)
			continue // Bỏ qua, chờ đợt quét sau
		}

		// 3. Update DB theo sự thật (Source of Truth) từ Gateway
		if isPaid {
			// Case Ghost Order: Đã thanh toán -> Update PAID
			order.Status = domain.OrderPaid
			log.Printf("Found GHOST ORDER %s -> Fixing to PAID", order.ID)
		} else {
			// Case thường: Chưa thanh toán (hoặc lỗi thật) -> Cancel luôn cho sạch DB
			order.Status = domain.OrderFailed
			log.Printf("Found ABANDONED ORDER %s -> Fixing to FAILED", order.ID)
		}

		// 4. Lưu vào DB (Không cần Transaction vì đây là lệnh sửa sai cuối cùng)
		// Lưu ý: UpdateOrderStatus cần TX, nhưng ở đây ta update lẻ tẻ.
		// Bạn có thể update `repo` để chấp nhận tx=nil hoặc cheat bằng cách begin 1 tx ngắn.
		// Tốt nhất: Gọi rw.orderRepo.UpdateOrderStatus(ctx, nil, &order) nếu repo hỗ trợ nil tx.
        // (Nếu repo bắt buộc tx, ta sẽ thêm logic begin tx ở đây).

        // Giả sử repo của bạn cần Tx, ta tạo 1 cái nho nhỏ
        // Note: Bạn cần pass db connection vào Worker nếu muốn start Tx.
        // Tạm thời để đơn giản hóa, hãy check xem repo của bạn có handle tx == nil không?
        // Nếu không, hãy sửa repo dòng: `execNode := tx; if tx == nil { execNode = r.db }`

        // Tạm thời mình giả định bạn đã sửa Repo để support tx == nil hoặc bạn sẽ tự inject DB vào Worker.
		// Dưới đây là code gọi giả định:
		tx, err := rw.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}

		defer tx.Rollback()
		order.Status = domain.OrderPaid
		_ = rw.orderRepo.UpdateOrderStatus(ctx, tx, &order)

		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}