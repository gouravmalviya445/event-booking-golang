package worker

import (
	"context"
	"log"
	"log/slog"
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()
	eventCollection := storage.Db.Collection("events")
	bookingCollection := storage.Db.Collection("bookings")

	session, err := storage.Client.StartSession(options.Session())
	defer session.EndSession(ctx)
	if err != nil {
		slog.Error("BgWorker: mongodb Session error", slog.String("err", err.Error()))
		return
	}
	slog.Info("background worker running...")

	_, err = session.WithTransaction(ctx, func(ctx context.Context) (any, error) {
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

		var booking models.Booking
		bookingResult := bookingCollection.
			FindOneAndUpdate(
				context.Background(),
				filter,
				update,
				options.FindOneAndUpdate().SetReturnDocument(options.After),
			)
		if bookingResult.Err() != nil {
			return nil, err
		}
		bookingResult.Decode(&booking)

		var event models.Event
		eventResult := eventCollection.
			FindOneAndUpdate(
				context.Background(),
				bson.M{"_id": booking.EventID},
				bson.M{"$inc": bson.M{"availableSeats": booking.Tickets}},
			)
		if eventResult.Err() != nil {
			return nil, err
		}
		eventResult.Decode(&event)

		return event.ID, nil
	})

	if err != nil {
		slog.Error("Error while releasing ticket in background", slog.String("err", err.Error()))
		return
	}

	slog.Info("Successfully release ticket in background")
}
