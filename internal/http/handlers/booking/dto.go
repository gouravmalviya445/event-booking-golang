package booking

// DTO -> Data transfer objects

type Booking struct {
	UserId string `json:"userId" validate:"required"`
	EventId string `json:"eventId" validate:"required"`

	// TODO: add payment later
}