package razorpay

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"os"

	rzp "github.com/razorpay/razorpay-go"
	"github.com/razorpay/razorpay-go/utils"
)

type Razorpay struct {
	Client *rzp.Client

	// Key and Secret
	ApiKey    string
	ApiSecret string
}

func New() *Razorpay {
	client := rzp.NewClient(os.Getenv("RAZORPAY_API_KEY"), os.Getenv("RAZORPAY_API_SECRET"))

	return &Razorpay{
		Client:    client,
		ApiKey:    os.Getenv("RAZORPAY_API_KEY"),
		ApiSecret: os.Getenv("RAZORPAY_API_SECRET"),
	}
}

func (r *Razorpay) CreateOrder(amount int64, currency string) (string, error) {
	data := map[string]any{
		"amount":   amount,
		"currency": currency,
		"receipt":  fmt.Sprintf("receipt_%v", rand.Text()),
	}
	
	body, err := r.Client.Order.Create(data, nil)
	if err != nil {
		slog.Error("Payment Order", slog.String("err", err.Error()))
		return "", fmt.Errorf("payment not initiated")
	}
	orderId := body["id"].(string)
	return orderId, nil
}

func (r *Razorpay) VerifyPayment(paymentId, orderId, signature string) bool {
	params := map[string]interface{}{
		"razorpay_order_id":   orderId,
		"razorpay_payment_id": paymentId,
	}

	return utils.VerifyPaymentSignature(params, signature, r.ApiSecret)
}

func (r *Razorpay) VerifyWebhook(webhookBody, signature string) bool {
	return utils.VerifyWebhookSignature(webhookBody, signature, r.ApiSecret)
}

func (r *Razorpay) RefundPayment(paymentId string, amount int) (string, error) {
	data := map[string]interface{}{"speed": "normal"}
	body, err := r.Client.Payment.Refund(paymentId, amount, data, nil)
	if err != nil {
		return "", fmt.Errorf("refund not initiated")
	}
	return body["id"].(string), nil
}
