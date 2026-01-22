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

		// create payment order
		orderId, err := payment.CreateOrder(body.Amount, body.Currency)
		if err != nil {
			response.WriteJson(
				w,
				http.StatusBadGateway,
				response.GeneralError(err),
			)
			return
		}

		// create booking with status pending
		pendingBooking, err := storage.CreatePendingBooking(userId, eventId, orderId, body.Currency, body.TotalTickets, body.Amount)
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
				"id": pendingBooking.RazorpayOrderID,
				"amount": pendingBooking.TotalPrice,
			}),
		)
	}
}

func VerifyPayment(storage storage.Storage, payment payment.Payment) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Info("Verifying order payment")

		// get req body
		var body BookingVerify
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil {
			if errors.Is(err, io.EOF) {
				response.WriteJson(
					w,
					http.StatusBadRequest,
					response.GeneralError(fmt.Errorf("Empty body")),
				)
			} else {
				response.WriteJson(w, http.StatusBadRequest, response.GeneralError(err))
			}
			return
		}

		// input validation
		err = validator.New().Struct(body)
		if err != nil {
			validationErr := err.(validator.ValidationErrors)
			response.WriteJson(w, http.StatusBadRequest, response.ValidationErr(validationErr))
			return
		}

		isValid := payment.VerifyPayment(body.RazorpayPaymentId, body.RazorpayOrderId, body.RazorpaySignature)

		if isValid == false {
			response.WriteJson(
				w,
				http.StatusBadRequest,
				response.GeneralError(fmt.Errorf("Invalid payment signature")),
			)
			return
		}

		booking, err := storage.UpdatePaymentIDAndSignature(body.RazorpayOrderId, body.RazorpayPaymentId, body.RazorpaySignature)
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
			http.StatusOK,
			response.GeneralResponse(map[string]any{"orderId": booking.RazorpayOrderID}),
		)
	}
}

func ConfirmBooking(storage storage.Storage, payment payment.Payment) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := io.ReadAll(r.Body)
		if err != nil {
			if errors.Is(err, io.EOF) {
				response.WriteJson(
					w,
					http.StatusBadRequest,
					response.GeneralError(fmt.Errorf("Empty Body")),
				)
			} else {
				response.WriteJson(
					w,
					http.StatusBadRequest,
					response.GeneralError(fmt.Errorf("please send valid body")),
				)
			}
			return
		}
		defer r.Body.Close()

		sign := r.Header.Get("X-Razorpay-Signature")

		if sign == "" {
			response.WriteJson(
				w,
				http.StatusBadRequest,
				response.GeneralError(fmt.Errorf("signature is required please check headers")),
			)
			return
		}

		// first verify webhook send r.Body in string formate
		isValid := payment.VerifyWebhook(string(b), sign)

		if isValid == false {
			response.WriteJson(
				w,
				http.StatusBadRequest,
				response.GeneralError(fmt.Errorf("invalid webhook signature")),
			)
			return
		}

		// read the required body
		var webhookBody BookingWebhook
		err = json.NewDecoder(r.Body).Decode(&webhookBody)
		if err != nil {
			response.WriteJson(
				w,
				http.StatusBadRequest,
				response.GeneralError(err),
			)
			return
		}

		booking, err := storage.UpdatePendingBooking(
			webhookBody.Event,
			webhookBody.Payload.Payment.Entity.OrderID,
			webhookBody.Payload.Payment.Entity.ID,
		)
		if err != nil {
			response.WriteJson(
				w,
				http.StatusInternalServerError,
				response.GeneralError(err),
			)
			return
		}

		// if booking status is expired then initiate refund
		if booking.Status == "expired" {
			_, err := payment.RefundPayment(booking.RazorpayPaymentID, booking.TotalPrice)
			if err != nil {
				response.WriteJson(
					w,
					http.StatusBadRequest,
					response.GeneralError(err),
				)
				return
			}
		}

		response.WriteJson(
			w,
			http.StatusAccepted,
			response.GeneralResponse("200 ok"),
		)
	}
}
