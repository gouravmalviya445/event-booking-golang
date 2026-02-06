package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gouravmalviya445/event-booking-golang/internal/config"
	"github.com/gouravmalviya445/event-booking-golang/internal/http/handlers/booking"
	"github.com/gouravmalviya445/event-booking-golang/internal/http/handlers/event"
	"github.com/gouravmalviya445/event-booking-golang/internal/service/payment/razorpay"
	"github.com/gouravmalviya445/event-booking-golang/internal/storage/mongodb"
	"github.com/gouravmalviya445/event-booking-golang/internal/worker"
)

func main() {
	// load config
	cfg := config.MustLoad()

	// database setup
	storage, err := mongodb.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	go worker.Worker(storage, time.Second*10, context.Background())

	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		// disconnect to database
		if err = storage.Close(ctx); err != nil {
			log.Fatal("Error in disconnecting from database", err)
		}
	}()

	// razorpay setup
	payment := razorpay.New()

	// setup router
	r := http.NewServeMux()

	// booking routes
	r.HandleFunc("POST /api/bookings/order", booking.Initiate(storage, payment))
	r.HandleFunc("POST /api/bookings/verify", booking.VerifyPayment(storage, payment))
	r.HandleFunc("POST /api/bookings/confirm", booking.ConfirmBooking(storage, payment))
	r.HandleFunc("GET /api/bookings/{id}", booking.CheckBookingStatus(storage))
	r.HandleFunc("GET /api/bookings/user/{id}", booking.GetUserBookings(storage))
	r.HandleFunc("GET /api/bookings", booking.GetAllBookings(storage))

	// event routes
	r.HandleFunc("GET /api/events", event.GetOrganizerEvents(storage))

	// setup server
	srv := http.Server{
		Addr:    cfg.HTTPServer.Addr,
		Handler: r,
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.Info("Server started", slog.String("address", srv.Addr))
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal("Failed to start server")
		}
	}()

	<-sig // block until signal not received

	// graceful shutdown
	slog.Info("Shutting down the server...")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Failed to shutdown server", slog.String("error", err.Error()))
	}

	slog.Info("Server shutdown successfully")
}
