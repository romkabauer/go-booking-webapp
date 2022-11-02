package model

type Booking struct {
	Id            string `json:"id"`
	CustomerName  string `json:"customer_name"`
	TicketsBooked uint   `json:"tickets_booked"`
	BookedAt      string `json:"booked_at"`
	UpdatedAt     string `json:"updated_at"`
	IsCanceled    bool   `json:"is_canceled"`
}
