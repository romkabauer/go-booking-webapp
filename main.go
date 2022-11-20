package main

import (
	"log"

	"github.com/gofiber/fiber/v2"

	"booking-webapp/database"
	"booking-webapp/router"
)

func main() {
	var err error
	database.UsersCollection, err = database.DBInit("users")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("UsersCollection initialized: %v\n", database.UsersCollection)
	database.ConferencesCollection, err = database.DBInit("conferences")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("ConferencesCollection initialized: %v\n", database.ConferencesCollection)

	app := fiber.New()

	router.SetupRoutes(app)

	app.Listen(":80")
}
