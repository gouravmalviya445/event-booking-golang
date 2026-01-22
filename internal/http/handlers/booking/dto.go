package booking

// DTO -> Data transfer objects

type BookingOrder struct {
	UserId       string `json:"userId" validate:"required"`
	EventId      string `json:"eventId" validate:"required"`
	Amount       int64  `json:"amount" validate:"required"`
	Currency     string `json:"currency" validate:"required"`
	TotalTickets int    `json:"totalTickets" validate:"required"`
}

type BookingVerify struct {
	RazorpayPaymentId string `json:"razorpay_payment_id" validate:"required"`
	RazorpayOrderId   string `json:"razorpay_order_id" validate:"required"`
	RazorpaySignature string `json:"razorpay_signature" validate:"required"`
}

// webhook data send by the razorpay in POST request to notify on event
type BookingWebhook struct {
	Event   string `json:"event" validate:"required"`
	Payload struct {
		Payment struct {
			Entity struct {
				ID       string `json:"id" validate:"required"`       // payment_id
				OrderID  string `json:"order_id" validate:"required"` // order_id
				Status   string `json:"status" validate:"required"`   // authorized / captured / failed
				Captured bool   `json:"captured" validate:"required"` // true only when captured
			} `json:"entity" validate:"required"`
		} `json:"payment" validate:"required"`
	} `json:"payload" validate:"required"`
}
