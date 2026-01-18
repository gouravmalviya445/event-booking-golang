package booking

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gouravmalviya445/event-booking-golang/internal/service/payment"
	"github.com/gouravmalviya445/event-booking-golang/internal/storage"
	"github.com/gouravmalviya445/event-booking-golang/internal/utils/response"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// create booking of an event
func Initiate(storage storage.Storage, payment payment.Payment) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Create booking order")

		// read input from r.body
		var body BookingOrder

		err := json.NewDecoder(r.Body).Decode(&body)
		if errors.Is(err, io.EOF) {
			// if body is empty
			response.WriteJson(w, http.StatusBadRequest, response.GeneralError(fmt.Errorf("empty body")))
			return
		}

		if err != nil {
			response.WriteJson(w, http.StatusBadRequest, response.GeneralError(err))
			return
		}

		// first validate the inputs
		if err := validator.New().Struct(body); err != nil {
			validationErrs := err.(validator.ValidationErrors)
			response.WriteJson(
				w,
				http.StatusBadRequest,
				response.ValidationErr(validationErrs),
			)
			return
		}

		// create razorpay payment order
		orderId, err := payment.CreateOrder(body.Amount, body.Currency)
		if err != nil {
			response.WriteJson(
				w,
				http.StatusBadGateway,
				response.GeneralError(fmt.Errorf("payment gateway error")),
			)
		}

		userId, err := bson.ObjectIDFromHex(body.UserId)
		if err != nil {
			response.WriteJson(
				w,
				http.StatusBadRequest,
				response.GeneralError(fmt.Errorf("invalid userId")),
			)
			return
		}
		eventId, err := bson.ObjectIDFromHex(body.EventId)
		if err != nil {
			response.WriteJson(
				w,
				http.StatusBadRequest,
				response.GeneralError(fmt.Errorf("invalid eventId")),
			)
			return
		}

		// create booking with status pending
		pendingBooking, err := storage.CreatePendingBooking(userId, eventId, orderId, body.Currency)
		if err != nil {
			response.WriteJson(
				w,
				http.StatusInternalServerError,
				response.GeneralError(err),
			)
			return
		}

		response.WriteJson(
			w,
			http.StatusCreated,
			response.GeneralResponse(map[string]any{
				"orderId": pendingBooking.RazorpayOrderID,
			}),
		)
	}
}
