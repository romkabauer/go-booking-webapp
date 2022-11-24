package handlers

import (
	"booking-webapp/database"
	"booking-webapp/errors"
	"booking-webapp/model"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetBookings(c *fiber.Ctx) error {
	if !isAdminRole(c) {
		return errors.RaisePermissionsError(c, "only admin can perform this operation")
	}

	conferences, dbErr := database.GetConferences(isAdminRole(c), "_id", c.Params("confId"))
	if dbErr != nil {
		return errors.RaiseInternalServerError(c, fmt.Sprintf("database error: %v", dbErr))
	}
	if len(conferences) == 0 {
		return errors.RaiseNotFoundError(c, fmt.Sprintf("conference %v not found", c.Params("confId")))
	}

	bookingsJson, jsonErr := json.MarshalIndent(conferences[0].Bookings, "", "	")
	if jsonErr != nil {
		return errors.RaiseInternalServerError(c, fmt.Sprintf("json serialization error: %v", jsonErr))
	}

	return c.SendString(string(bookingsJson))
}

func GetBooking(c *fiber.Ctx) error {
	conferences, dbErr := database.GetConferences(true, "_id", c.Params("confId"))
	if dbErr != nil {
		return errors.RaiseInternalServerError(c, fmt.Sprintf("database error: %v", dbErr))
	}
	if len(conferences) == 0 {
		return errors.RaiseNotFoundError(c, fmt.Sprintf("conference %v not found", c.Params("confId")))
	}

	for _, booking := range conferences[0].Bookings {
		if booking.Id.Hex() == c.Params("bookingId") {
			bookingJson, jsonErr := json.MarshalIndent(booking, "", "	")
			if jsonErr != nil {
				return errors.RaiseInternalServerError(c, fmt.Sprintf("json serialization error: %v", jsonErr))
			}
			return c.SendString(string(bookingJson))
		}
	}
	return errors.RaiseNotFoundError(c,
		fmt.Sprintf("booking %v not found for conference %v", c.Params("bookingId"), c.Params("confId")))
}

func CreateBooking(c *fiber.Ctx) error {
	conferences, dbErr := database.GetConferences(true, "_id", c.Params("confId"))
	if dbErr != nil {
		return errors.RaiseInternalServerError(c, fmt.Sprintf("database error: %v", dbErr))
	}
	if len(conferences) == 0 {
		return errors.RaiseNotFoundError(c, fmt.Sprintf("conference %v not found", c.Params("confId")))
	}
	conference := conferences[0]

	newBooking := new(model.Booking)
	if jsonErr := c.BodyParser(newBooking); jsonErr != nil {
		return errors.RaiseBadRequestError(c, fmt.Sprintf("unacceptable booking parameters: %v", jsonErr))
	}
	newBooking.Id = primitive.NewObjectID()
	newBooking.CustomerName = strings.TrimSpace(newBooking.CustomerName)
	newBooking.IsCanceled = false

	currentTime := time.Now().Format(time.RFC3339)
	newBooking.BookedAt = currentTime
	newBooking.UpdatedAt = currentTime

	customerNameErr := customerNameValidation(newBooking.CustomerName)
	numberOfTicketsErr := ticketsNumberValidation(newBooking.TicketsBooked, conference.RemainingTickets)

	if customerNameErr != nil {
		return errors.RaiseBadRequestError(c,
			fmt.Sprintf("incorrect input for booking parameters: %v", customerNameErr))
	} else if numberOfTicketsErr != nil {
		return errors.RaiseBadRequestError(c,
			fmt.Sprintf("incorrect input for booking parameters: %v", numberOfTicketsErr))
	}

	insertErr := database.InsertBooking(conference, *newBooking)
	if insertErr != nil {
		return errors.RaiseInternalServerError(c, fmt.Sprintf("database error: %v", insertErr))
	}

	newBookingJson, jsonErr := json.MarshalIndent(newBooking, "", "	")
	if jsonErr != nil {
		return errors.RaiseInternalServerError(c, fmt.Sprintf("json serialization error: %v", jsonErr))
	}

	return c.SendString(string(newBookingJson))
}

func UpdateBooking(c *fiber.Ctx) error {
	conferences, dbErr := database.GetConferences(true, "_id", c.Params("confId"))
	if dbErr != nil {
		return errors.RaiseInternalServerError(c, fmt.Sprintf("database error: %v", dbErr))
	}
	if len(conferences) == 0 {
		return errors.RaiseNotFoundError(c, fmt.Sprintf("conference %v not found", c.Params("confId")))
	}
	conference := conferences[0]

	var currentBooking model.Booking = model.Booking{}
	var currentBookingIndex int = -1
	for bookingIndex, booking := range conference.Bookings {
		if booking.Id.Hex() == c.Params("bookingId") && !booking.IsCanceled {
			currentBooking = booking
			currentBookingIndex = bookingIndex
			break
		} else if booking.Id.Hex() == c.Params("bookingId") && booking.IsCanceled {
			return errors.RaiseBadRequestError(c, "cannot update canceled booking")
		}
	}

	if currentBookingIndex == -1 {
		return errors.RaiseNotFoundError(c,
			fmt.Sprintf("booking %v not found for conference %v", c.Params("bookingId"), c.Params("confId")))
	}

	updatedBooking := new(model.Booking)
	if jsonErr := c.BodyParser(updatedBooking); jsonErr != nil {
		return errors.RaiseBadRequestError(c, fmt.Sprintf("unacceptable booking parameters: %v", jsonErr))
	}
	updatedBooking.Id = currentBooking.Id
	updatedBooking.CustomerName = strings.TrimSpace(updatedBooking.CustomerName)
	updatedBooking.BookedAt = currentBooking.BookedAt
	updatedBooking.IsCanceled = currentBooking.IsCanceled

	deltaTickets := int(updatedBooking.TicketsBooked) - int(currentBooking.TicketsBooked)

	reqPathParts := strings.Split(c.OriginalURL(), "/")
	var validationErr error = nil
	// adjust validation according to the request path, e.g. do only name validation if name update occure
	if reqPathParts[len(reqPathParts)-1] == "name" {
		updatedBooking.TicketsBooked = currentBooking.TicketsBooked
		validationErr = customerNameValidation(updatedBooking.CustomerName)
	} else if reqPathParts[len(reqPathParts)-1] == "tickets" {
		updatedBooking.CustomerName = currentBooking.CustomerName
		if deltaTickets > 0 {
			validationErr = ticketsNumberValidation(
				updatedBooking.TicketsBooked,
				conference.RemainingTickets+currentBooking.TicketsBooked)
		}
	} else {
		nameValidationErr := customerNameValidation(updatedBooking.CustomerName)
		if nameValidationErr != nil {
			validationErr = nameValidationErr
		} else if deltaTickets > 0 {
			validationErr = ticketsNumberValidation(
				updatedBooking.TicketsBooked,
				conference.RemainingTickets+currentBooking.TicketsBooked)
		}
	}
	if validationErr != nil {
		return errors.RaiseBadRequestError(c,
			fmt.Sprintf("incorrect input for booking parameters: %v", validationErr))
	}

	insertErr := database.InsertBooking(conference, *updatedBooking, uint(currentBookingIndex))
	if insertErr != nil {
		return errors.RaiseInternalServerError(c, fmt.Sprintf("database error: %v", insertErr))
	}

	updatedBookingJson, jsonErr := json.MarshalIndent(updatedBooking, "", "	")
	if jsonErr != nil {
		return errors.RaiseInternalServerError(c, fmt.Sprintf("json serialization error: %v", jsonErr))
	}

	return c.SendString(string(updatedBookingJson))
}

func CancelBooking(c *fiber.Ctx) error {
	conferences, dbErr := database.GetConferences(true, "_id", c.Params("confId"))
	if dbErr != nil {
		return errors.RaiseInternalServerError(c, fmt.Sprintf("database error: %v", dbErr))
	}
	if len(conferences) == 0 {
		return errors.RaiseNotFoundError(c, fmt.Sprintf("conference %v not found", c.Params("confId")))
	}
	conference := conferences[0]

	for bookingIndex, booking := range conference.Bookings {
		if booking.Id.Hex() == c.Params("bookingId") && !booking.IsCanceled {
			booking.IsCanceled = true

			insertErr := database.InsertBooking(conference, booking, uint(bookingIndex))
			if insertErr != nil {
				return errors.RaiseInternalServerError(c, fmt.Sprintf("database error: %v", insertErr))
			}

			bookingJson, jsonErr := json.MarshalIndent(booking, "", "	")
			if jsonErr != nil {
				return errors.RaiseInternalServerError(c, fmt.Sprintf("json serialization error: %v", jsonErr))
			}

			return c.SendString(string(bookingJson))
		} else if booking.Id.Hex() == c.Params("bookingId") && booking.IsCanceled {
			return errors.RaiseBadRequestError(c, "booking is already canceled")
		}
	}

	return errors.RaiseNotFoundError(c,
		fmt.Sprintf("booking %v not found for conference %v", c.Params("bookingId"), c.Params("confId")))
}

func customerNameValidation(customerName string) error {
	if len(customerName) < 2 {
		return fmt.Errorf("name is too short")
	} else if !strings.Contains(customerName, " ") {
		return fmt.Errorf("last name is missing, try format 'firstName LastName'")
	}
	return nil
}

func ticketsNumberValidation(ticketsToBook uint, remainingTickets uint) error {
	if ticketsToBook == 0 {
		return fmt.Errorf("cannot book 0 tickets")
	} else if ticketsToBook > remainingTickets {
		return fmt.Errorf("only %v tickets left for the conference, overbooking is not supported", remainingTickets)
	}
	return nil
}
