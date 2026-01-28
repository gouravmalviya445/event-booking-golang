package storage

import (
	"github.com/gouravmalviya445/event-booking-golang/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Storage interface {
	CreatePendingBooking(userId, eventId bson.ObjectID, orderId, currency string, totalTickets int, amount int) (*models.Booking, error)
	UpdatePaymentIDAndSignature(orderId, paymentId, signature string) (*models.Booking, error)
	UpdatePendingBooking(event, orderId, paymentId string) (*models.Booking, error)
	GetBookingStatus(orderId string) (string, error)

	GetBookings(userId bson.ObjectID) (*[]models.BookingWithEvent, error)

	GetEventsOfOrganizer(organizerId bson.ObjectID) (*[]models.Event, error)
}
