package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type Event struct {
	ID             bson.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Title          string        `bson:"title" json:"title"`
	Description    string        `bson:"description" json:"description"`
	Location       string        `bson:"location" json:"location"`
	Date           time.Time     `bson:"date" json:"date"`
	Price          float64       `bson:"price" json:"price"`
	TotalSeats     int           `bson:"totalSeats" json:"totalSeats"`
	AvailableSeats int           `bson:"availableSeats" json:"availableSeats"`
	Status         string        `bson:"status" json:"status"`
	Category       string        `bson:"category" json:"category"`
	Image          string        `bson:"image" json:"image"`
	Organizer      bson.ObjectID `bson:"organizer" json:"organizer"`
	CreatedAt      time.Time     `bson:"createdAt" json:"createdAt"`
	UpdatedAt      time.Time     `bson:"updatedAt" json:"updatedAt"`
}
