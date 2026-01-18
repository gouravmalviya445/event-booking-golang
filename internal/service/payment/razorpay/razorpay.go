package razorpay

import (
	"os"

	rzp "github.com/razorpay/razorpay-go"
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
