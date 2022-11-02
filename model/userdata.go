package model

type UserData struct {
	Login          string `json:"login"`
	HashedPassword string `json:"password_hash"`
}
