package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Booking struct {
	ID      bson.ObjectID `bson:"_id,omitempty" json:"_id"`
	UserID  bson.ObjectID `bson:"userId" json:"userId"`
	EventID bson.ObjectID `bson:"eventId" json:"eventId"`

	Tickets    int    `bson:"tickets" json:"tickets"`
	TotalPrice int    `bson:"totalPrice" json:"totalPrice"`
	Status     string `bson:"status" json:"status"`

	RazorpayOrderID   string `bson:"razorpayOrderId" json:"razorpay_order_id"`
	RazorpayPaymentID string `bson:"razorpayPaymentId" json:"razorpay_payment_id"`
	RazorpaySignature string `bson:"razorpaySignature" json:"razorpay_signature"`

	Currency string `bson:"currency" json:"currency"`

	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
	UpdatedAt time.Time `bson:"updatedAt" json:"updatedAt"`
	ExpiredAt time.Time `bson:"expiredAt" json:"expiredAt"`
}

type BookingWithEvent struct {
	ID         bson.ObjectID `bson:"_id" json:"_id"`
	Tickets    int           `bson:"tickets" json:"tickets"`
	TotalPrice int           `bson:"totalPrice," json:"totalPrice"`
	CreatedAt  time.Time     `bson:"createdAt" json:"createdAt"`
	Event      struct {
		Date     time.Time `bson:"date" json:"date"`
		Category string `bson:"category" json:"category"`
	} `bson:"event" json:"event"`
}
