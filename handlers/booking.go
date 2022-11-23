package handlers

import (
	"booking-webapp/database"
	"booking-webapp/model"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetBookings(c *fiber.Ctx) error {
	if !isAdminRole(c) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "lack of permissions",
			"data":    nil})
	}

	conferences, dberr := database.GetConferences(isAdminRole(c), "_id", c.Params("confId"))
	if dberr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while database call",
			"data":    fmt.Sprint(dberr)})
	}

	if len(conferences) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "conference not found",
			"data":    nil})
	}

	bookingsJson, err := json.MarshalIndent(conferences[0].Bookings, "", "	")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while sending bookings info to client",
			"data":    err})
	}

	return c.SendString(string(bookingsJson))
}

func GetBooking(c *fiber.Ctx) error {
	conferences, dberr := database.GetConferences(true, "_id", c.Params("confId"))
	if dberr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while database call",
			"data":    fmt.Sprint(dberr)})
	}

	if len(conferences) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "conference not found",
			"data":    nil})
	}

	for _, booking := range conferences[0].Bookings {
		if booking.Id.Hex() == c.Params("bookingId") {
			bookingJson, err := json.MarshalIndent(booking, "", "	")
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"status":  "error",
					"message": "server side problem occured while sending booking info to client",
					"data":    err})
			}

			return c.SendString(string(bookingJson))
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"status":  "error",
		"message": "booking not found",
		"data":    fmt.Errorf("no booking with id %v for conference id %v", c.Params("bookingId"), c.Params("confId"))})
}

func CreateBooking(c *fiber.Ctx) error {
	newBooking := new(model.Booking)

	if err := c.BodyParser(newBooking); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "incorrect input for booking parameters",
			"data":    err})
	}
	newBooking.CustomerName = strings.TrimSpace(newBooking.CustomerName)

	newBooking.Id = primitive.NewObjectID()
	currentTime := time.Now().Format(time.RFC3339)

	newBooking.BookedAt = currentTime
	newBooking.UpdatedAt = currentTime
	newBooking.IsCanceled = false

	conferences, dberr := database.GetConferences(true, "_id", c.Params("confId"))
	if dberr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while database call",
			"data":    fmt.Sprint(dberr)})
	}

	if len(conferences) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "conference not found",
			"data":    nil})
	}

	customerNameValidation := customerNameValidation(newBooking.CustomerName)
	numberOfTicketsValidation := ticketsNumberValidation(newBooking.TicketsBooked, conferences[0].RemainingTickets)

	if customerNameValidation != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "incorrect input for booking parameters",
			"data":    fmt.Sprint(customerNameValidation)})
	}
	if numberOfTicketsValidation != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "incorrect input for booking parameters",
			"data":    fmt.Sprint(numberOfTicketsValidation)})
	}

	updatedConf := conferences[0]

	updatedConf.RemainingTickets = updatedConf.RemainingTickets - newBooking.TicketsBooked
	updatedConf.Bookings = append(updatedConf.Bookings, *newBooking)

	updateErr := database.UpdateCollectionItem(
		updatedConf.Id,
		updatedConf,
		database.ConferencesCollection)
	if updateErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "db error while updating bookings info",
			"data":    fmt.Sprint(updateErr)})
	}

	newBookingJson, err := json.MarshalIndent(newBooking, "", "	")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while sending booking info to client",
			"data":    fmt.Sprint(err)})
	}

	return c.SendString(string(newBookingJson))
}

func UpdateBooking(c *fiber.Ctx) error {
	conferences, dberr := database.GetConferences(true, "_id", c.Params("confId"))
	if dberr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while database call",
			"data":    fmt.Sprint(dberr)})
	}

	if len(conferences) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "conference not found",
			"data":    nil})
	}

	var booking model.Booking = model.Booking{}
	var bookingIndex int = -1
	for prevBookingIndex, prevBooking := range conferences[0].Bookings {
		if prevBooking.Id.Hex() == c.Params("bookingId") && !prevBooking.IsCanceled {
			booking = prevBooking
			bookingIndex = prevBookingIndex
			break
		} else if prevBooking.Id.Hex() == c.Params("bookingId") && prevBooking.IsCanceled {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "cannot update canceled booking",
				"data":    nil})
		}
	}

	if bookingIndex == -1 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "booking not found",
			"data":    fmt.Errorf("no booking with id %v for conference id %v", c.Params("bookingId"), c.Params("confId"))})
	}

	updatedBooking := new(model.Booking)

	if err := c.BodyParser(updatedBooking); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "incorrect input for booking parameters",
			"data":    err})
	}
	updatedBooking.CustomerName = strings.TrimSpace(updatedBooking.CustomerName)
	deltaTickets := int(booking.TicketsBooked) - int(updatedBooking.TicketsBooked)

	reqPathParts := strings.Split(c.OriginalURL(), "/")
	var validationErr error = nil
	// adjust validation according to the request path, e.g. do only name validation if name update occure
	if reqPathParts[len(reqPathParts)-1] == "name" {
		updatedBooking.TicketsBooked = booking.TicketsBooked
		validationErr = customerNameValidation(updatedBooking.CustomerName)
	} else if reqPathParts[len(reqPathParts)-1] == "tickets" {
		updatedBooking.CustomerName = booking.CustomerName
		if deltaTickets > 0 {
			validationErr = ticketsNumberValidation(uint(deltaTickets), conferences[0].RemainingTickets)
		}
	} else {
		nameValidationErr := customerNameValidation(updatedBooking.CustomerName)
		ticketsValidationErr := ticketsNumberValidation(uint(deltaTickets), conferences[0].RemainingTickets)
		if nameValidationErr != nil {
			validationErr = nameValidationErr
		} else {
			validationErr = ticketsValidationErr
		}
	}
	if validationErr != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "incorrect input for booking parameters",
			"data":    fmt.Sprint(validationErr)})
	}

	updatedBooking.Id = booking.Id
	updatedBooking.BookedAt = booking.BookedAt
	updatedBooking.UpdatedAt = time.Now().Format(time.RFC3339)
	updatedBooking.IsCanceled = booking.IsCanceled

	updatedBookingJson, err := json.MarshalIndent(updatedBooking, "", "	")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while sending booking info to client",
			"data":    err})
	}

	updatedConf := conferences[0]
	updatedConf.Bookings[bookingIndex] = *updatedBooking
	updatedConf.RemainingTickets = uint(int(updatedConf.RemainingTickets) + deltaTickets)

	updateErr := database.UpdateCollectionItem(
		updatedConf.Id,
		updatedConf,
		database.ConferencesCollection)
	if updateErr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "db error while updating bookings info",
			"data":    fmt.Sprint(updateErr)})
	}

	return c.SendString(string(updatedBookingJson))
}

func CancelBooking(c *fiber.Ctx) error {
	conferences, dberr := database.GetConferences(true, "_id", c.Params("confId"))
	if dberr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while database call",
			"data":    fmt.Sprint(dberr)})
	}

	if len(conferences) == 0 {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"status":  "error",
			"message": "conference not found",
			"data":    nil})
	}

	conference := conferences[0]

	for bookingIndex, booking := range conference.Bookings {
		if booking.Id.Hex() == c.Params("bookingId") && !booking.IsCanceled {
			booking.UpdatedAt = time.Now().Format(time.RFC3339)
			booking.IsCanceled = true
			conference.RemainingTickets += booking.TicketsBooked
			conference.Bookings[bookingIndex] = booking

			updateErr := database.UpdateCollectionItem(
				conference.Id,
				conference,
				database.ConferencesCollection)
			if updateErr != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"status":  "error",
					"message": "db error while updating bookings info",
					"data":    fmt.Sprint(updateErr)})
			}

			bookingJson, err := json.MarshalIndent(booking, "", "	")
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"status":  "error",
					"message": "server side problem occured while sending booking info to client",
					"data":    err})
			}

			return c.SendString(string(bookingJson))
		} else if booking.Id.Hex() == c.Params("bookingId") && booking.IsCanceled {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "booking is already canceled",
				"data":    nil})
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"status":  "error",
		"message": "booking not found",
		"data":    fmt.Sprintf("no booking with id %v for conference id %v", c.Params("bookingId"), c.Params("confId"))})
}

func customerNameValidation(customerName string) error {
	if len(customerName) < 2 {
		return errors.New("name is too short")
	} else if !strings.Contains(customerName, " ") {
		return errors.New("last name is missing, try format 'firstName LastName'")
	}
	return nil
}

func ticketsNumberValidation(ticketsToBook uint, remainingTickets uint) error {
	if ticketsToBook == 0 {
		return errors.New("cannot book 0 tickets")
	} else if ticketsToBook > remainingTickets {
		return fmt.Errorf("only %v tickets left for the conference, overbooking is not supported", remainingTickets)
	}
	return nil
}
