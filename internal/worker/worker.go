package worker

import (
	"context"
	"log"
	"time"

	"github.com/gouravmalviya445/event-booking-golang/internal/models"
	"github.com/gouravmalviya445/event-booking-golang/internal/storage/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func Worker(storage *mongodb.MongoDB, duration time.Duration, ctx context.Context) {
	ticker := time.NewTicker(duration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("worker finished its job")
		case <-ticker.C:
			releaseTicket(storage)
		}
	}
}

func releaseTicket(storage *mongodb.MongoDB) {
	eventCollection := storage.Db.Collection("events")
	bookingCollection := storage.Db.Collection("bookings")

	filter := bson.M{
		"status": "pending",
		"expiredAt": bson.M{
			"$lt": time.Now().UTC(),
		},
	}
	update := bson.M{
		"$set": bson.M{
			"status":    "expired",
			"updatedAt": time.Now().UTC(),
		},
	}

	log.Println("background worker...")
	var booking models.Booking
	bookingCollection.FindOneAndUpdate(context.Background(), filter, update, options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&booking)

	var event models.Event
	eventCollection.FindOneAndUpdate(context.Background(), bson.M{"_id": booking.EventID}, bson.M{"$inc": bson.M{"availableSeats": booking.Tickets}}).Decode(&event)

}
