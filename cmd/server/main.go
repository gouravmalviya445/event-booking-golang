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
)

func main() {
	// load config
	cfg := config.MustLoad()

	// database setup

	// setup router
	r := http.NewServeMux()

	r.HandleFunc("POST /api/bookings", booking.CreateBooking())

	// setup server
	srv := http.Server{
		Addr:    cfg.HTTPServer.Addr,
		Handler: r,
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	go func(){
		slog.Info("Server started", slog.String("address", srv.Addr))
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal("Failed to start server")
		}
	}()

	<-sig // block until signal not received

	// graceful shutdown
	slog.Info("Shutting down the server...")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 5)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		slog.Error("Failed to shutdown server", slog.String("error", err.Error()))
	}

	slog.Info("Server shutdown successfully")
}
