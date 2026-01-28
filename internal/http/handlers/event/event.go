package event

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gouravmalviya445/event-booking-golang/internal/storage"
	"github.com/gouravmalviya445/event-booking-golang/internal/utils/response"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func GetOrganizerEvents(storage storage.Storage) http.HandlerFunc { 
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Get organizer event")
		id := r.URL.Query().Get("organizerId")
		if id == "" {
			response.WriteJson(
				w, http.StatusBadRequest, response.GeneralError(fmt.Errorf("organizer id is required")),
			)
			return 
		}

		organizerId, err := bson.ObjectIDFromHex(id)
		if err != nil {
			response.WriteJson(
				w, http.StatusBadRequest, response.GeneralError(fmt.Errorf("organizer id is invalid")),
			)
			return
		}

		events, err := storage.GetEventsOfOrganizer(organizerId)
		if err != nil {
			response.WriteJson(
				w, http.StatusBadRequest, response.GeneralError(err),
			)
			return
		}

		var totalEvents int = 0
		var totalRevenue int = 0
		var totalTicketsSold int = 0
		var totalActiveEvents int = 0
		for _, e := range *events {
			totalEvents++
			totalRevenue = totalRevenue + (e.Price * (e.TotalSeats - e.AvailableSeats))
			totalTicketsSold = totalTicketsSold + (e.TotalSeats - e.AvailableSeats)
			if e.Date.After(time.Now().UTC()) {
				totalActiveEvents++;	
			}
		}

		response.WriteJson(
			w, http.StatusOK, response.GeneralResponse(map[string]any{
				"totalEvents": totalEvents,
				"totalRevenue": totalRevenue,
				"totalTicketsSold": totalTicketsSold,
				"totalActiveEvents": totalActiveEvents,
				"events": events,
			}),
		)
	}
}