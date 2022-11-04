package handlers

import (
	"booking-webapp/database"
	"booking-webapp/model"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
)

func GetConferences(c *fiber.Ctx) error {
	conferences, readerr := database.ReadLocalDB()
	if readerr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while reading conferences info from database",
			"data":    readerr})
	}

	if !isAdminRole(c) {
		conferences = cleanBookingsData(conferences)
	}

	conferencesJson, err := json.MarshalIndent(conferences, "", "	")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while loading conferences info",
			"data":    err})
	}

	return c.SendString(string(conferencesJson))
}

func GetConference(c *fiber.Ctx) error {
	conference, geterr := database.GetConference(c.Params("id"))
	if geterr := database.HandleGetConferenceError(geterr, c); geterr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while reading conference info from database",
			"data":    geterr})
	}

	if !isAdminRole(c) {
		conference = cleanBookingsData([]model.Conference{conference})[0]
	}

	conferenceJson, err := json.MarshalIndent(conference, "", "	")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while sending conference info to client",
			"data":    err})
	}
	return c.SendString(string(conferenceJson))
}

func CreateNewConference(c *fiber.Ctx) error {
	if !isAdminRole(c) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "lack of permissions",
			"data":    nil})
	}

	newConf := new(model.Conference)
	if err := c.BodyParser(newConf); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "incorrect input for conferrence parameters",
			"data":    err})
	}
	newConf.ConferenceName = strings.TrimSpace(newConf.ConferenceName)

	validationErr := validateConferenceInfoInput(*newConf, true)
	if validationErr != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "incorrect input for conferrence parameters",
			"data":    fmt.Sprint(validationErr)})
	}

	newUuid, _ := uuid.NewRandom()
	newConf.Id = strings.Replace(newUuid.String(), "-", "", -1)
	newConf.RemainingTickets = newConf.TotalTickets
	newConf.Bookings = []model.Booking{}

	newConfJson, err := json.MarshalIndent(newConf, "", "	")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while sending conference info to client",
			"data":    err})
	}

	commiterr := database.CommitConferenceToLocalDB(*newConf)
	if commiterr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "error while saving transaction result to the database",
			"data":    commiterr})
	}

	return c.SendString(string(newConfJson))
}

func UpdateConference(c *fiber.Ctx) error {
	if !isAdminRole(c) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "lack of permissions",
			"data":    nil})
	}

	conference, geterr := database.GetConference(c.Params("id"))
	if geterr := database.HandleGetConferenceError(geterr, c); geterr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while reading conference info from database",
			"data":    geterr})
	}

	updatedConf := new(model.Conference)

	if err := c.BodyParser(updatedConf); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "incorrect input for conferrence parameters",
			"data":    err})
	}
	updatedConf.ConferenceName = strings.TrimSpace(updatedConf.ConferenceName)

	reqPathParts := strings.Split(c.OriginalURL(), "/")
	var validationErr error = nil
	// adjust validation according to the request path, e.g. do only name validation if name update occure
	if reqPathParts[len(reqPathParts)-1] == "name" {
		updatedConf.TotalTickets = conference.TotalTickets
		validationErr = isValidConferenceName(updatedConf.ConferenceName, false)
	} else if reqPathParts[len(reqPathParts)-1] == "tickets" {
		updatedConf.ConferenceName = conference.ConferenceName
		validationErr = isValidConferenceTotalTickets(*updatedConf, false)
	} else {
		validationErr = validateConferenceInfoInput(*updatedConf, false)
	}
	if validationErr != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "incorrect input for conferrence parameters",
			"data":    fmt.Sprint(validationErr)})
	}

	updatedConf.Id = conference.Id
	updatedConf.RemainingTickets = updatedConf.TotalTickets - database.GetTotalBookings(conference)
	updatedConf.Bookings = conference.Bookings

	updatedConfJson, err := json.MarshalIndent(updatedConf, "", "	")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while sending conference info to client",
			"data":    err})
	}

	commiterr := database.CommitConferenceToLocalDB(*updatedConf)
	if commiterr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "error while saving transaction result to the database",
			"data":    commiterr})
	}

	return c.SendString(string(updatedConfJson))
}

func DeleteConference(c *fiber.Ctx) error {
	if !isAdminRole(c) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "lack of permissions",
			"data":    nil})
	}

	conferences, dbreaderr := database.ReadLocalDB()
	if dbreaderr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Server side problem occured while reading conferences info from database.")
	}
	confId := c.Params("id")

	for confIndex, conference := range conferences {
		if conference.Id == confId {
			conferences = append(conferences[:confIndex], conferences[confIndex+1:]...)
			database.CommitConferencesToLocalDB(conferences)
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"status":  "success",
				"message": "conference deleted",
				"data":    fmt.Sprintf("conference with id %v was deleted", confId)})
		}
	}

	return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
		"status":  "error",
		"message": "conference not found",
		"data":    fmt.Sprintf("no conference with id %v to delete", confId)})
}

func validateConferenceInfoInput(conf model.Conference, isNew bool) error {
	nameValidationErr := isValidConferenceName(conf.ConferenceName, isNew)
	totalTicketsValidationErr := isValidConferenceTotalTickets(conf, isNew)
	if nameValidationErr != nil {
		return fmt.Errorf("incorrect input for conferrence name: %v", nameValidationErr)
	}
	if totalTicketsValidationErr != nil {
		return fmt.Errorf("incorrect input for conferrence total number of tickets: %v", totalTicketsValidationErr)
	}

	return nil
}

func isValidConferenceName(name string, isNew bool) error {
	if len(name) < 2 {
		return errors.New("conference name is too short")
	} else if isNew {
		nameExists, err := ifConferenceNameAlreadyExist(name)
		if err != nil {
			return err
		}
		if nameExists {
			return errors.New("conference name already exist")
		}
	}
	return nil
}

func ifConferenceNameAlreadyExist(name string) (bool, error) {
	conferences, dbreaderr := database.ReadLocalDB()
	if dbreaderr != nil {
		return false, fmt.Errorf("server side problem occured while reading conferences info from database")
	}

	for _, conference := range conferences {
		if conference.ConferenceName == name {
			return true, nil
		}
	}
	return false, nil
}

func isValidConferenceTotalTickets(conf model.Conference, isNew bool) error {
	if conf.TotalTickets == 0 {
		return errors.New("conference cannot have zero tickets for distribution")
	}
	if !isNew {
		totalBookings := database.GetTotalBookings(conf)
		if conf.TotalTickets < totalBookings {
			return fmt.Errorf("cannot assign %v as total tickets, %v tickets already booked", conf.TotalTickets, totalBookings)
		}
		return nil
	}
	return nil
}

func isAdminRole(c *fiber.Ctx) bool {
	token := c.Locals("identity").(*jwt.Token)
	claims := token.Claims.(jwt.MapClaims)
	return claims["role"].(string) == "admin"
}

func cleanBookingsData(conferences []model.Conference) []model.Conference {
	for confIndex, conference := range conferences {
		conference.Bookings = []model.Booking{}
		conferences[confIndex] = conference
	}

	return conferences
}
