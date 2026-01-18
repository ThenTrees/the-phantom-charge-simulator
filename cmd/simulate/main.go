package main

import (
	"context"
	"fmt"
	"log"
	"the-phantom-charge/internal/database"
	"the-phantom-charge/internal/infrastructure/payment"
	"the-phantom-charge/internal/repo"
	"the-phantom-charge/internal/service"
	"the-phantom-charge/internal/worker"
	"time"
)

func main() {
	ctx := context.Background()
	db := database.NewPostgres()

	orderRepo := repo.NewOrderRepo(db)
	paymentRepo := repo.NewPaymentRepo(db)
	paymentGateway := payment.NewPaymentGateway()
	orderService := service.NewOrderService(db, orderRepo, paymentRepo, paymentGateway)

    fmt.Println("--- STARTING SIMULATION (20 ORDERS) ---")
	for i := 0; i < 20; i++ {
		// 1. Create
		order, err := orderService.CreateOrder(ctx)
		if err != nil {
			log.Printf("Create Failed: %v", err)
			continue
		}

		// 2. Checkout (Có thể lỗi mạng)
		fmt.Printf("[%d] Processing Order %s ... ", i+1, order.ID)
		_, err = orderService.Checkout(ctx, order.ID)

        // Log kết quả Checkout
		if err != nil {
			fmt.Printf("FAILED: %v\n", err)
		} else {
			fmt.Printf("SUCCESS\n")
		}

		// 3. QUAN TRỌNG: Query lại DB để xem trạng thái thực tế
        // Nếu Checkout Failed (Timeout) mà DB vẫn là PAID -> Ghost Order (Logic cũ, đã fix)
        // Nếu Checkout Failed (Timeout) mà DB là PENDING -> Ghost Order Case mới (Mất tiền, không có đơn).
		freshOrder, _ := orderRepo.FindById(ctx, order.ID)
		fmt.Printf("    -> DB Status: %s\n", freshOrder.Status)
        fmt.Println("---------------------------------------------------")
        time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(2 * time.Second)

	worker := worker.NewReconciliationWorker(db, orderRepo, paymentGateway, 1*time.Second)
	go worker.Run(ctx)

	time.Sleep(10 * time.Second)
}