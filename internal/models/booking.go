package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Booking struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"_id"`
	UserID     primitive.ObjectID `bson:"userId" json:"userId"`
	EventID    primitive.ObjectID `bson:"eventId" json:"eventId"`
	Tickets    int                `bson:"tickets" json:"tickets"`
	TotalPrice float64            `bson:"totalPrice" json:"totalPrice"`
	Status     string             `bson:"status" json:"status"`
	CreatedAt  time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt  time.Time          `bson:"updatedAt" json:"updatedAt"`
}
