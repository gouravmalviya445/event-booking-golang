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

// create booking with status "pending"
func (m *MongoDB) CreatePendingBooking(userId, eventId bson.ObjectID, orderId, currency string) (*models.Booking, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	eventCollection := m.Db.Collection("events")
	bookingCollection := m.Db.Collection("bookings")

	filter := bson.M{
		"_id":            eventId,
		"availableSeats": bson.M{"$gt": 0},
	}
	update := bson.M{"$inc": bson.M{"availableSeats": -1}}

	var event models.Event // event model
	result := eventCollection.FindOneAndUpdate(ctx, filter, update)

	if result.Err() != nil {
		if errors.Is(result.Err(), mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("event sold out or not found")
		}
		return nil, result.Err()
	}

	err := result.Decode(&event)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	booking := models.Booking{
		ID:              bson.NewObjectID(),
		UserID:          userId,
		EventID:         eventId,
		Status:          "pending",
		Tickets:         1,
		RazorpayOrderID: orderId,
		TotalPrice:      event.Price * 1,
		Currency:        currency,
		CreatedAt:       now,
		UpdatedAt:       now,
		ExpiredAt:       time.Now().Add(time.Minute * 5).UTC(), // for reserving a ticket
	}

	_, err = bookingCollection.InsertOne(ctx, booking)
	if err != nil {
		slog.Error(err.Error())
		return nil, fmt.Errorf("failed to create booking %w", err)
	}
	return &booking, nil
}

// update Payment ID and Signature
func (m *MongoDB) UpdatePaymentIDAndSignature(orderId, paymentId, signature string) (*models.Booking, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	bookingCollection := m.Db.Collection("bookings")

	// find whose status is pending
	// bcz what if webhook will notify first
	filter := bson.M{
		"razorpayOrderId": orderId,
		"status":          "pending",
		"expiredAt":       bson.M{"$gt": time.Now().UTC()},
	}
	update := bson.M{
		"$set": bson.M{
			"razorpayPaymentId": paymentId,
			"razorpaySignature": signature,
		},
	}

	var booking models.Booking

	result := bookingCollection.FindOneAndUpdate(ctx, filter, update)
	if result.Err() != nil {
		if errors.Is(result.Err(), mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("booking is expired or not found")
		}
		return nil, result.Err()
	}

	// decode booking
	err := result.Decode(&booking)
	if err != nil {
		return nil, err
	}

	return &booking, nil
}

// disconnect
func (m *MongoDB) Close(ctx context.Context) error {
	return m.Client.Disconnect(ctx)
}
