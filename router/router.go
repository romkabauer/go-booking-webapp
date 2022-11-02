package router

import (
	"booking-webapp/handlers"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func SetupRoutes(app *fiber.App) {
	api := app.Group("/", logger.New())
	api.Get("/hello", handlers.GetHello)

	//Login
	login := api.Group("/login")
	login.Post("/", handlers.Login)

	//Conference
	conference := api.Group("/conference")
	conference.Get("/", handlers.GetConferences)
	conference.Get("/:id", handlers.GetConference)
	conference.Post("/", handlers.CreateNewConference)
	conference.Put("/:id", handlers.UpdateConference)
	conference.Patch("/:id/name", handlers.UpdateConference)
	conference.Patch("/:id/tickets", handlers.UpdateConference)
	conference.Delete("/:id", handlers.DeleteConference)

	//Booking
	booking := conference.Group("/:confId/booking")
	booking.Get("/", handlers.GetBookings)
	booking.Get("/:bookingId", handlers.GetBooking)
	booking.Post("/", handlers.CreateBooking)
	booking.Put("/:bookingId", handlers.UpdateBooking)
	booking.Patch("/:bookingId/name", handlers.UpdateBooking)
	booking.Patch("/:bookingId/tickets", handlers.UpdateBooking)
	booking.Patch("/:bookingId/cancel", handlers.CancelBooking)
}
