package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type Conference struct {
	Id               primitive.ObjectID `json:"_id" bson:"_id"`
	ConferenceName   string             `json:"conference_name" bson:"conference_name"`
	TotalTickets     uint               `json:"total_tickets" bson:"total_tickets"`
	RemainingTickets uint               `json:"remaining_tickets" bson:"remaining_tickets"`
	Bookings         []Booking          `json:"bookings" bson:"bookings"`
}
