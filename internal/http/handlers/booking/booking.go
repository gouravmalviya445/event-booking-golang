package booking

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/gouravmalviya445/event-booking-golang/internal/utils/response"
)

// create booking of an event
func Create() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Info("create a booking")

		// read input from r.body
		var booking Booking

		err := json.NewDecoder(r.Body).Decode(&booking)
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
		if err := validator.New().Struct(booking); err != nil {
			validationErrs := err.(validator.ValidationErrors)
			response.WriteJson(
				w,
				http.StatusBadRequest,
				response.ValidationErr(validationErrs),
			)
			return
		}
	}
}
