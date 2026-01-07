package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Event struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"_id,omitempty"`
	Title          string             `bson:"title" json:"title" validate:"required,min=3,max=100"`
	Description    string             `bson:"description" json:"description"`
	Location       string             `bson:"location" json:"location" validate:"required"`
	Date           time.Time          `bson:"date" json:"date" validate:"required"`
	Price          float64            `bson:"price" json:"price" validate:"min=0"`
	TotalSeats     int                `bson:"totalSeats" json:"totalSeats" validate:"required,min=1"`
	AvailableSeats int                `bson:"availableSeats" json:"availableSeats" validate:"min=0"`
	Status         string             `bson:"status" json:"status" validate:"required,oneof=active cancelled"`
	Category       string             `bson:"category" json:"category" validate:"required"`
	Image          string             `bson:"image" json:"image" validate:"required,url"`
	Organizer      primitive.ObjectID `bson:"organizer" json:"organizer" validate:"required"`
	CreatedAt      time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt      time.Time          `bson:"updatedAt" json:"updatedAt"`
}
