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

// create booking with status "pending"
func (m *MongoDB) CreatePendingBooking(userId, eventId bson.ObjectID, orderId, currency string, totalTickets int, amount int) (*models.Booking, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	// collections
	eventCollection := m.Db.Collection("events")
	bookingCollection := m.Db.Collection("bookings")

	txnOpts := options.Transaction().SetReadConcern(readconcern.Majority())
	sessionOpts := options.Session().SetDefaultTransactionOptions(txnOpts)

	// Starts a session on the client
	session, err := m.Client.StartSession(sessionOpts)
	if err != nil {
		return nil, fmt.Errorf("error while starting db session")
	}

	// Defers ending the session after the transaction is committed or ended
	defer session.EndSession(ctx)

	result, err := session.WithTransaction(ctx, func(ctx context.Context) (any, error) {
		filter := bson.M{
			"_id":            eventId,
			"availableSeats": bson.M{"$gt": 0},
		}
		update := bson.M{"$inc": bson.M{"availableSeats": -totalTickets}}

		// find event if exist and temporarily reserve tickets
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
			Tickets:         totalTickets,
			RazorpayOrderID: orderId,
			TotalPrice:      amount,
			Currency:        currency,
			CreatedAt:       now,
			UpdatedAt:       now,
			ExpiredAt:       time.Now().Add(time.Minute * 5).UTC(), // for reserving a ticket
		}

		// create booking with status "pending" and give an expiry of 5 minutes
		// if user will not pay in this time expire the booking and release the tickets
		_, err = bookingCollection.InsertOne(ctx, booking)
		if err != nil {
			slog.Error(err.Error())
			return nil, fmt.Errorf("failed to create booking %w", err)
		}
		return &booking, nil
	})

	return result.(*models.Booking), nil
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

// update booking status
func (m *MongoDB) UpdatePendingBooking(event, orderId, paymentId string) (*models.Booking, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	bookingCollection := m.Db.Collection("bookings")
	eventCollection := m.Db.Collection("events")

	bookingFilter := bson.M{"razorpayOrderId": orderId}
	result := bookingCollection.FindOne(ctx, bookingFilter)
	if result.Err() != nil {
		return nil, fmt.Errorf("booking not found")
	}

	// decode booking
	var booking models.Booking
	_ = result.Decode(&booking)

	switch event {
	case "payment.authorized":
		slog.Info("payment is being authorized")
		return &booking, nil
	case "payment.captured":
		slog.Info("payment is being captured")
		if booking.Status == "expired" || booking.Status == "success" || booking.Status == "refunded" {
			return &booking, nil
		}

		isBookingExpired := time.Now().After(booking.ExpiredAt)
		if isBookingExpired {
			bookingFilter := bson.M{
				"razorpayOrderId": orderId,
				"status":          "pending",
			}
			bookingUpdate := bson.M{
				"$set": bson.M{
					"razorpayPaymentId": paymentId,
					"status":            "expired",
					"updatedAt":         time.Now().UTC(),
				},
			}
			result := bookingCollection.
				FindOneAndUpdate(
					ctx, bookingFilter, bookingUpdate,
					options.FindOneAndUpdate().SetReturnDocument(options.After),
				)
			if result.Err() != nil {
				return nil, result.Err()
			}
			var updatedBooking models.Booking
			_ = result.Decode(&updatedBooking)

			// release tickets
			eventFilter := bson.M{"_id": booking.EventID}
			eventUpdate := bson.M{
				"$inc": bson.M{"availableSeats": booking.Tickets},
				"$set": bson.M{"updatedAt": time.Now().UTC()},
			}
			result = eventCollection.FindOneAndUpdate(ctx, eventFilter, eventUpdate)
			if result.Err() != nil {
				return nil, fmt.Errorf("error while releasing event tickets")
			}

			return &updatedBooking, nil
		}

		if booking.Status == "pending" {
			filter := bson.M{"_id": booking.ID, "status": booking.Status}
			update := bson.M{"$set": bson.M{"status": "success", "updatedAt": time.Now().UTC()}}
			result := bookingCollection.FindOneAndUpdate(
				ctx, filter, update,
				options.FindOneAndUpdate().SetReturnDocument(options.After),
			)
			if result.Err() != nil {
				return nil, result.Err()
			}
			var updatedBooking models.Booking
			_ = result.Decode(&updatedBooking)
			return &updatedBooking, nil
		}
	}
	return nil, nil
}

// get booking status
func (m *MongoDB) GetBookingStatus(orderId string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	bookingCollection := m.Db.Collection("bookings")

	var booking models.Booking
	result := bookingCollection.FindOne(ctx, bson.M{"razorpayOrderId": orderId})

	if result.Err() != nil {
		if errors.Is(result.Err(), mongo.ErrNoDocuments) {
			return "", fmt.Errorf("booking not found")
		}
		return "", result.Err()
	}

	err := result.Decode(&booking)
	if err != nil {
		return "", err
	}

	return booking.Status, nil
}

// get bookings with userid
func (m *MongoDB) GetBookings(userId bson.ObjectID) (*[]models.BookingWithEvent, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	bookingCollection := m.Db.Collection("bookings")

	matchStage := bson.D{
		{Key: "$match", Value: bson.D{
			{Key: "status", Value: "success"},
			{Key: "userId", Value: userId},
		}},
	}

	lookupStage := bson.D{
		{Key: "$lookup", Value: bson.D{
			{Key: "from", Value: "events"},
			{Key: "localField", Value: "eventId"},
			{Key: "foreignField", Value: "_id"},
			{Key: "as", Value: "event"},
		}},
	}

	addFieldStage := bson.D{
		{Key: "$addFields", Value: bson.D{
			{Key: "event", Value: bson.D{
				{Key: "$first", Value: "$event"},
			}},
		}},
	}

	projectStage := bson.D{
		{Key: "$project", Value: bson.D{
			{Key: "totalPrice", Value: 1},
			{Key: "tickets", Value: 1},
			{Key: "createdAt", Value: 1},
			{Key: "event", Value: bson.D{
				{Key: "date", Value: 1},
				{Key: "category", Value: 1},
				{Key: "price", Value: 1},
				{Key: "title", Value: 1},
			}},
		}},
	}

	cursor, err := bookingCollection.Aggregate(
		ctx,
		mongo.Pipeline{
			matchStage,
			lookupStage,
			addFieldStage,
			projectStage,
		},
	)
	if err != nil {
		slog.Error("Aggregation Pipeline", slog.String("err", err.Error()))
		return nil, err
	}

	var bookings []models.BookingWithEvent
	if err = cursor.All(ctx, &bookings); err != nil {
		fmt.Println(err)
		return nil, fmt.Errorf("error while decoding booking with event data")
	}

	return &bookings, nil
}

// get organizer events
func (m MongoDB) GetEventsOfOrganizer(organizerId bson.ObjectID) (*[]models.Event, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()
	eventCollection := m.Db.Collection("events")

	
	cursor, err := eventCollection.Find(ctx, bson.M{"organizer": organizerId})
	if err != nil  {
		slog.Error("Aggregation Pipeline", slog.String("err", err.Error()))
		return nil, err
	}

	var events []models.Event
	cursor.All(ctx, &events)
	
	return &events, nil
}

// disconnect
func (m *MongoDB) Close(ctx context.Context) error {
	return m.Client.Disconnect(ctx)
}
