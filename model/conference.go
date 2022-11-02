package model

type Conference struct {
	Id               string    `json:"id"`
	ConferenceName   string    `json:"conference_name"`
	TotalTickets     uint      `json:"total_tickets"`
	RemainingTickets uint      `json:"remaining_tickets"`
	Bookings         []Booking `json:"bookings"`
}
