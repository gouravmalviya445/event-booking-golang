package payment

type Payment interface {
	CreateOrder(amount int, currency string) (string, error)
	VerifyPayment(paymentId, orderId, signature string) bool
	VerifyWebhook(webhookBody, signature string) bool
	RefundPayment(paymentId string, amount int) (string, error)
}
