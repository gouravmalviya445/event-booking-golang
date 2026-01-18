package razorpay

import (
	"crypto/rand"
	"fmt"
	"os"

	rzp "github.com/razorpay/razorpay-go"
	"github.com/razorpay/razorpay-go/utils"
)

type Razorpay struct {
	Client *rzp.Client
}

// Key and Secret
var apiKey = os.Getenv("RAZORPAY_KEY_ID")
var apiSecret = os.Getenv("RAZORPAY_KEY_SECRET")

func New() *Razorpay {
	client := rzp.NewClient(apiKey, apiSecret)

	return &Razorpay{
		Client: client,
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

	return utils.VerifyPaymentSignature(params, signature, apiSecret)
}
