package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Booking struct {
	ID         bson.ObjectID `bson:"_id,omitempty" json:"_id"`
	UserID     bson.ObjectID `bson:"userId" json:"userId"`
	EventID    bson.ObjectID `bson:"eventId" json:"eventId"`
	Tickets    int           `bson:"tickets" json:"tickets"`
	TotalPrice float64       `bson:"totalPrice" json:"totalPrice"`
	Status     string        `bson:"status" json:"status"`
	CreatedAt  time.Time     `bson:"createdAt" json:"createdAt"`
	UpdatedAt  time.Time     `bson:"updatedAt" json:"updatedAt"`
}
