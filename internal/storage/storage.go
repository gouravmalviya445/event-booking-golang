package storage

type Storage interface {
	CreateBooking(userId, eventId string) (string, error)
}
