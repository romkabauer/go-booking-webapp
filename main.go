package main

import (
	"github.com/gofiber/fiber/v2"

	"booking-webapp/router"
)

func main() {
	app := fiber.New()

	router.SetupRoutes(app)

	app.Listen(":80")
}
