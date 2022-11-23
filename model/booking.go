package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type Booking struct {
	Id            primitive.ObjectID `json:"_id" bson:"_id"`
	CustomerName  string             `json:"customer_name" bson:"customer_name"`
	TicketsBooked uint               `json:"tickets_booked" bson:"tickets_booked"`
	BookedAt      string             `json:"booked_at" bson:"booked_at"`
	UpdatedAt     string             `json:"updated_at" bson:"updated_at"`
	IsCanceled    bool               `json:"is_canceled" bson:"is_canceled"`
}
