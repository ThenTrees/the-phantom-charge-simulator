ISSUSE:
User clicks "Pay" on the Frontend.
Frontend sends POST /api/checkout with {order_id, amount}.
Backend validates order_id.
Backend calls FastPay HTTP API (POST /v1/payment/charge) synchronously.
FastPay returns 200 OK (Success).
Backend updates the database: UPDATE orders SET status = 'PAID' WHERE id = ?.
Backend returns 200 OK to Frontend.

Duplicate Charges: Customer Support is flooded with tickets. "I was charged TWICE for the same order!"
Ghost Orders: Finance reconciled the books and found 50 transactions where FastPay took the money, but our database still shows the order as PENDING. The users are angry because they paid but didn't get the item.

SOLUTION:
- Khi yêu cầu fastPay thanh toán sẽ gửi kèm Idempotency key trong payload. FastPay check đã có request với key này chưa, nếu có thì trả về response cũ, nếu chưa thì tạo request mới.
=> tránh được việc user click nút thanh toán nhiều lần dẫn đến việc bị trừ tiền nhiều lần.

- Sẽ có 1 cronjob chạy để check các đơn hàng đã thanh toán nhưng còn trạng thái pending => update trạng thái đơn hàng thành paid.
=> tránh được việc user thanh toán nhưng không nhận được hàng.