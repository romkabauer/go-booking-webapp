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
	"github.com/google/uuid"
)

func GetBookings(c *fiber.Ctx) error {
	if !isAdminRole(c) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "lack of permissions",
			"data":    nil})
	}

	conference, geterr := database.GetConference(c.Params("confId"))
	if geterr := database.HandleGetConferenceError(geterr, c); geterr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while reading bokings info from database",
			"data":    geterr})
	}

	bookingsJson, err := json.MarshalIndent(conference.Bookings, "", "	")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while sending bookings info to client",
			"data":    err})
	}

	return c.SendString(string(bookingsJson))
}

func GetBooking(c *fiber.Ctx) error {
	conference, geterr := database.GetConference(c.Params("confId"))
	if geterr := database.HandleGetConferenceError(geterr, c); geterr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while reading bookings info from database",
			"data":    geterr})
	}

	for _, booking := range conference.Bookings {
		if booking.Id == c.Params("bookingId") {
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

	newUuid, _ := uuid.NewRandom()
	newBooking.Id = strings.Replace(newUuid.String(), "-", "", -1)
	currentTime := time.Now().Format(time.RFC3339)

	newBooking.BookedAt = currentTime
	newBooking.UpdatedAt = currentTime
	newBooking.IsCanceled = false

	conference, geterr := database.GetConference(c.Params("confId"))
	if geterr := database.HandleGetConferenceError(geterr, c); geterr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while reading bookings info from database",
			"data":    geterr})
	}

	customerNameValidation := customerNameValidation(newBooking.CustomerName)
	numberOfTicketsValidation := ticketsNumberValidation(newBooking.TicketsBooked, conference.RemainingTickets)

	if customerNameValidation != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "incorrect input for booking parameters",
			"data":    customerNameValidation})
	}
	if numberOfTicketsValidation != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "incorrect input for booking parameters",
			"data":    numberOfTicketsValidation})
	}

	conference.RemainingTickets = conference.RemainingTickets - newBooking.TicketsBooked
	conference.Bookings = append(conference.Bookings, *newBooking)
	commiterr := database.CommitConferenceToLocalDB(conference)
	if commiterr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while saving booking info to the database",
			"data":    commiterr})
	}

	newBookingJson, err := json.MarshalIndent(newBooking, "", "	")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while sending booking info to client",
			"data":    err})
	}

	return c.SendString(string(newBookingJson))
}

func UpdateBooking(c *fiber.Ctx) error {
	conference, geterr := database.GetConference(c.Params("confId"))
	if geterr := database.HandleGetConferenceError(geterr, c); geterr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while reading bookings info from database",
			"data":    geterr})
	}

	var booking model.Booking = model.Booking{}
	var bookingIndex int = -1
	for prevBookingIndex, prevBooking := range conference.Bookings {
		if prevBooking.Id == c.Params("bookingId") && !prevBooking.IsCanceled {
			booking = prevBooking
			bookingIndex = prevBookingIndex
			break
		} else if prevBooking.Id == c.Params("bookingId") && prevBooking.IsCanceled {
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

	reqPathParts := strings.Split(c.OriginalURL(), "/")
	var validationErr error = nil
	// adjust validation according to the request path, e.g. do only name validation if name update occure
	if reqPathParts[len(reqPathParts)-1] == "name" {
		updatedBooking.TicketsBooked = booking.TicketsBooked
		validationErr = customerNameValidation(updatedBooking.CustomerName)
	} else if reqPathParts[len(reqPathParts)-1] == "tickets" {
		updatedBooking.CustomerName = booking.CustomerName
		validationErr = ticketsNumberValidation(updatedBooking.TicketsBooked, conference.RemainingTickets)
	} else {
		validationErr = validateBookingInfoInput(*updatedBooking, conference.RemainingTickets)
	}
	if validationErr != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "incorrect input for booking parameters",
			"data":    validationErr})
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

	conference.Bookings[bookingIndex] = *updatedBooking
	conference.RemainingTickets = conference.TotalTickets - database.GetTotalBookings(conference)
	commiterr := database.CommitConferenceToLocalDB(conference)
	if commiterr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while saving booking info to the database",
			"data":    commiterr})
	}

	return c.SendString(string(updatedBookingJson))
}

func CancelBooking(c *fiber.Ctx) error {
	conference, geterr := database.GetConference(c.Params("confId"))
	if geterr := database.HandleGetConferenceError(geterr, c); geterr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while reading bookings info from database",
			"data":    geterr})
	}

	for bookingIndex, booking := range conference.Bookings {
		if booking.Id == c.Params("bookingId") && !booking.IsCanceled {
			booking.UpdatedAt = time.Now().Format(time.RFC3339)
			booking.IsCanceled = true
			conference.RemainingTickets += booking.TicketsBooked
			conference.Bookings[bookingIndex] = booking
			commiterr := database.CommitConferenceToLocalDB(conference)
			if commiterr != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"status":  "error",
					"message": "server side problem occured while saving booking info to the database",
					"data":    commiterr})
			}

			bookingJson, err := json.MarshalIndent(booking, "", "	")
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"status":  "error",
					"message": "server side problem occured while sending booking info to client",
					"data":    err})
			}

			return c.SendString(string(bookingJson))
		} else if booking.Id == c.Params("bookingId") && booking.IsCanceled {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"status":  "error",
				"message": "booking is already canceled",
				"data":    nil})
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"status":  "error",
		"message": "booking not found",
		"data":    fmt.Errorf("no booking with id %v for conference id %v", c.Params("bookingId"), c.Params("confId"))})
}

func validateBookingInfoInput(booking model.Booking, remainingTickets uint) error {
	nameValidationErr := customerNameValidation(booking.CustomerName)
	ticketsValidationErr := ticketsNumberValidation(booking.TicketsBooked, remainingTickets)
	if nameValidationErr != nil {
		return nameValidationErr
	}
	if ticketsValidationErr != nil {
		return ticketsValidationErr
	}

	return nil
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
