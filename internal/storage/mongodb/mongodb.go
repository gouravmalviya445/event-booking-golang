package mongodb

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/gouravmalviya445/event-booking-golang/internal/config"
	"github.com/gouravmalviya445/event-booking-golang/internal/models"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
)

type MongoDB struct {
	Client *mongo.Client
	Db     *mongo.Database

	// locks map an eventId -> *sync.mutex
	// locks sync.Map // this is the special map that is useful for concurrent task
}

// create an instance of MongoDB struct
func New(cfg *config.Config) (*MongoDB, error) {
	if cfg.Database.URI == "" {
		slog.Error("mongodb URI is not provided")
		return nil, fmt.Errorf("mongodb URI is not provided")
	}

	// configure: set server api version to 1
	serverAPI := options.ServerAPI(options.ServerAPIVersion1).SetDeprecationErrors(true)

	// Define the options for mongodb client
	opts := options.Client().ApplyURI(cfg.Database.URI).SetServerAPIOptions(serverAPI)

	if os.Getenv("ENV") == "development" {
		opts.SetDirect(true) // force to connect with standalone
	}

	// connect to mongodb with ClientOptions
	client, err := mongo.Connect(opts)
	if err != nil {
		slog.Error("mongodb connection failed")
		return nil, fmt.Errorf("mongodb connection failed err: %s", err.Error())
	}

	// ping mongodb to confirm a successful connection
	ctxPing, cancelPing := context.WithTimeout(context.Background(), time.Second*5)
	defer cancelPing()
	if err = client.Ping(ctxPing, nil); err != nil {
		return nil, fmt.Errorf("mongodb ping failed: %s", err.Error())
	}

	slog.Info("MongoDB connected successfully")

	return &MongoDB{
		Client: client,
		Db:     client.Database(cfg.DbName),
	}, nil
}

// HELPER METHOD
// func (m *MongoDB) getLockForEvent(eventId bson.ObjectID) *sync.Mutex {
// 	// If User A is booking a ticket for "Coldplay," User B has to wait to book a ticket for "IPL Final."
// 	// So Instead of locking the whole database, we should only lock the specific Event ID
// 	// User A locks "Coldplay", User B locks "IPL Final" simultaneously

// 	// LoadOrStore tries to load the lock. If it doesn't exist, it saves a new one.
// 	lock, _ := m.locks.LoadOrStore(eventId, &sync.Mutex{})

// 	// We must cast the empty interface{} back to a Mutex pointer
// 	return lock.(*sync.Mutex)
// }

// implement storage interface "/internal/storage/storage.go"
func (m *MongoDB) CreateBooking(userId, eventId bson.ObjectID) (*models.Booking, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	eventCollection := m.Db.Collection("events")
	bookingCollection := m.Db.Collection("bookings")

	tnxOpts := options.
		Transaction().
		SetReadConcern(readconcern.Majority())
	sessionOpts := options.
		Session().
		SetDefaultTransactionOptions(tnxOpts)

	// Starts a session on the client
	session, err := m.Client.StartSession(sessionOpts)
	if err != nil {
		return nil, err
	}

	defer session.EndSession(ctx)

	// start transaction
	result, err := session.WithTransaction(ctx, func(ctx context.Context) (any, error) {
		filter := bson.M{
			"_id":            eventId,
			"availableSeats": bson.M{"$gt": 0},
		}
		update := bson.M{"$inc": bson.M{"availableSeats": -1}}

		var event models.Event // event model
		err = eventCollection.FindOneAndUpdate(ctx, filter, update).Decode(&event)
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return nil, fmt.Errorf("event sold out or not found")
			}
			return nil, err
		}

		now := time.Now().UTC()
		booking := models.Booking{
			ID:         bson.NewObjectID(),
			UserID:     userId,
			EventID:    eventId,
			Status:     "confirmed", // TODO: first payment then book
			Tickets:    1,           // TODO: add multiple ticket buying option
			TotalPrice: event.Price * 1,
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		_, err = bookingCollection.InsertOne(ctx, booking)
		if err != nil {
			slog.Error(err.Error())
			return nil, fmt.Errorf("failed to create booking %w", err)
		}
		return &booking, nil
	})

	if err != nil {
		return nil, err
	}

	return result.(*models.Booking), nil
}

// disconnect
func (m *MongoDB) Close(ctx context.Context) error {
	return m.Client.Disconnect(ctx)
}
