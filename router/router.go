package router

import (
	"booking-webapp/handlers"
	"booking-webapp/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func SetupRoutes(app *fiber.App) {
	api := app.Group("/", logger.New())
	api.Get("/hello/:id", middleware.Authorize(), handlers.GetHello)

	//Login
	login := api.Group("/login")
	login.Post("/", handlers.Login)

	//Conference
	conference := api.Group("/conference")
	conference.Get("/", middleware.Authorize(), handlers.GetConferences)
	conference.Get("/:id", middleware.Authorize(), handlers.GetConference)
	conference.Post("/", middleware.Authorize(), handlers.CreateNewConference)
	conference.Put("/:id", middleware.Authorize(), handlers.UpdateConference)
	conference.Patch("/:id/name", middleware.Authorize(), handlers.UpdateConference)
	conference.Patch("/:id/tickets", middleware.Authorize(), handlers.UpdateConference)
	conference.Delete("/:id", middleware.Authorize(), handlers.DeleteConference)

	//Booking
	booking := conference.Group("/:confId/booking")
	booking.Get("/", middleware.Authorize(), handlers.GetBookings)
	booking.Get("/:bookingId", handlers.GetBooking)
	booking.Post("/", handlers.CreateBooking)
	booking.Put("/:bookingId", handlers.UpdateBooking)
	booking.Patch("/:bookingId/name", handlers.UpdateBooking)
	booking.Patch("/:bookingId/tickets", handlers.UpdateBooking)
	booking.Patch("/:bookingId/cancel", handlers.CancelBooking)
}
