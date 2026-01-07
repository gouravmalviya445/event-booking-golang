package storage

import (
	"github.com/gouravmalviya445/event-booking-golang/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Storage interface {
	CreateBooking(userId, eventId bson.ObjectID) (*models.Booking, error)
}
