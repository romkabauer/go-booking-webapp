package model

import "go.mongodb.org/mongo-driver/bson/primitive"

type UserData struct {
	Id             primitive.ObjectID `json:"_id" bson:"_id"`
	Login          string             `json:"login" bson:"login,omitempty"`
	HashedPassword string             `json:"password_hash" bson:"password_hash,omitempty"`
	Role           string             `json:"role" bson:"role,omitempty"`
}
