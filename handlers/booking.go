package handlers

import (
	"booking-webapp/database"
	"booking-webapp/model"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

func GetBookings(c *fiber.Ctx) error {
	conference, geterr := database.GetConference(c.Params("confId"))
	if geterr := database.HandleGetConferenceError(geterr, c); geterr != nil {
		return geterr
	}

	bookingsJson, err := json.MarshalIndent(conference.Bookings, "", "	")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Server side problem occured while loading bookings info.")
	}

	return c.SendString(string(bookingsJson))
}

func GetBooking(c *fiber.Ctx) error {
	conference, geterr := database.GetConference(c.Params("confId"))
	if geterr := database.HandleGetConferenceError(geterr, c); geterr != nil {
		return geterr
	}

	for _, booking := range conference.Bookings {
		if booking.Id == c.Params("bookingId") {
			bookingJson, err := json.MarshalIndent(booking, "", "	")
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "Server side problem occured while loading bookings info.")
			}

			return c.SendString(string(bookingJson))
		}
	}

	return fiber.NewError(fiber.StatusNotFound, fmt.Sprintf("No booking with Id %v for conference %v", c.Params("bookingId"), c.Params("confId")))
}

func CreateBooking(c *fiber.Ctx) error {
	newBooking := new(model.Booking)

	if err := c.BodyParser(newBooking); err != nil {
		return fiber.NewError(
			fiber.StatusBadRequest,
			fmt.Sprintf("Incorrect input for conferrence parameters. Details:\n%v", err))
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
		return geterr
	}

	customerNameValidation := customerNameValidation(newBooking.CustomerName)
	numberOfTicketsValidation := ticketsNumberValidation(newBooking.TicketsBooked, conference.RemainingTickets)

	if customerNameValidation != nil {
		return fiber.NewError(
			fiber.StatusBadRequest,
			fmt.Sprintf("Incorrect input for customer name. Details:\n%v", customerNameValidation))
	}
	if numberOfTicketsValidation != nil {
		return fiber.NewError(
			fiber.StatusBadRequest,
			fmt.Sprintf("Incorrect input for number of tickets for booking. Details:\n%v", numberOfTicketsValidation))
	}

	conference.RemainingTickets = conference.RemainingTickets - newBooking.TicketsBooked
	conference.Bookings = append(conference.Bookings, *newBooking)
	commiterr := database.CommitConferenceToLocalDB(conference)
	if commiterr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Error during writing booking to the DB.")
	}

	newBookingJson, err := json.MarshalIndent(newBooking, "", "	")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Server side problem occured while loading new booking info.")
	}

	return c.SendString(string(newBookingJson))
}

func UpdateBooking(c *fiber.Ctx) error {
	conference, geterr := database.GetConference(c.Params("confId"))
	if geterr := database.HandleGetConferenceError(geterr, c); geterr != nil {
		return geterr
	}

	var booking model.Booking = model.Booking{}
	var bookingIndex uint = 0
	for prevBookingIndex, prevBooking := range conference.Bookings {
		if prevBooking.Id == c.Params("bookingId") && !prevBooking.IsCanceled {
			booking = prevBooking
			bookingIndex = uint(prevBookingIndex)
			break
		} else if prevBooking.Id == c.Params("bookingId") && prevBooking.IsCanceled {
			return fiber.NewError(fiber.StatusBadRequest, "cannot update canceled booking")
		}
	}

	updatedBooking := new(model.Booking)

	if err := c.BodyParser(updatedBooking); err != nil {
		return fiber.NewError(
			fiber.StatusBadRequest,
			fmt.Sprintf("Incorrect input for booking parameters. Details:\n%v", err))
	}
	updatedBooking.CustomerName = strings.TrimSpace(updatedBooking.CustomerName)

	reqPathParts := strings.Split(c.OriginalURL(), "/")
	var validationErr error = nil
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
		return validationErr
	}

	updatedBooking.Id = booking.Id
	updatedBooking.BookedAt = booking.BookedAt
	updatedBooking.UpdatedAt = time.Now().Format(time.RFC3339)
	updatedBooking.IsCanceled = booking.IsCanceled

	updatedBookingJson, err := json.MarshalIndent(updatedBooking, "", "	")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Server side problem occured while loading booking info.")
	}

	conference.Bookings[bookingIndex] = *updatedBooking
	conference.RemainingTickets = conference.TotalTickets - database.GetTotalBookings(conference)
	commiterr := database.CommitConferenceToLocalDB(conference)
	if commiterr != nil {
		return commiterr
	}

	return c.SendString(string(updatedBookingJson))
}

func CancelBooking(c *fiber.Ctx) error {
	conference, geterr := database.GetConference(c.Params("confId"))
	if geterr := database.HandleGetConferenceError(geterr, c); geterr != nil {
		return geterr
	}

	for bookingIndex, booking := range conference.Bookings {
		if booking.Id == c.Params("bookingId") && !booking.IsCanceled {
			booking.UpdatedAt = time.Now().Format(time.RFC3339)
			booking.IsCanceled = true
			conference.RemainingTickets += booking.TicketsBooked
			conference.Bookings[bookingIndex] = booking
			commiterr := database.CommitConferenceToLocalDB(conference)
			if commiterr != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "error while commiting changes to the database")
			}

			bookingJson, err := json.MarshalIndent(booking, "", "	")
			if err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "server side problem occured while loading bookings info.")
			}

			return c.SendString(string(bookingJson))
		} else if booking.Id == c.Params("bookingId") && booking.IsCanceled {
			return fiber.NewError(fiber.StatusBadRequest, "booking is already canceled")
		}
	}

	return fiber.NewError(fiber.StatusNotFound, fmt.Sprintf("booking with id %v not found", c.Params("bookingId")))
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
		return fiber.NewError(fiber.StatusBadRequest, "name is too short")
	} else if !strings.Contains(customerName, " ") {
		return fiber.NewError(fiber.StatusBadRequest, "last name is missing, try format 'firstName LastName'")
	}
	return nil
}

func ticketsNumberValidation(ticketsToBook uint, remainingTickets uint) error {
	if ticketsToBook == 0 {
		return fiber.NewError(fiber.StatusBadRequest, "cannot book 0 tickets")
	} else if ticketsToBook > remainingTickets {
		return fiber.NewError(
			fiber.StatusBadRequest,
			fmt.Sprintf("only %v tickets left for the conference, overbooking is not supported", remainingTickets))
	}
	return nil
}
