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
		return fiber.NewError(fiber.StatusInternalServerError, "Server side problem occured while reading conferences info from database.")
	}

	if !isAdminRole(c) {
		conferences = cleanBookingsData(conferences)
	}

	conferencesJson, err := json.MarshalIndent(conferences, "", "	")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Server side problem occured while loading conferences info.")
	}

	return c.SendString(string(conferencesJson))
}

func GetConference(c *fiber.Ctx) error {
	conference, geterr := database.GetConference(c.Params("id"))
	if geterr := database.HandleGetConferenceError(geterr, c); geterr != nil {
		return geterr
	}

	if !isAdminRole(c) {
		conference = cleanBookingsData([]model.Conference{conference})[0]
	}

	conferenceJson, err := json.MarshalIndent(conference, "", "	")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Server side problem occured while loading conferences info.")
	}
	return c.SendString(string(conferenceJson))
}

func CreateNewConference(c *fiber.Ctx) error {
	newConf := new(model.Conference)
	if err := c.BodyParser(newConf); err != nil {
		return fiber.NewError(
			fiber.StatusBadRequest,
			fmt.Sprintf("Incorrect input for conferrence parameters. Details:\n%v", err))
	}
	newConf.ConferenceName = strings.TrimSpace(newConf.ConferenceName)

	validationErr := validateConferenceInfoInput(*newConf, true)
	if validationErr != nil {
		return validationErr
	}

	newUuid, _ := uuid.NewRandom()
	newConf.Id = strings.Replace(newUuid.String(), "-", "", -1)
	newConf.RemainingTickets = newConf.TotalTickets
	newConf.Bookings = []model.Booking{}

	newConfJson, err := json.MarshalIndent(newConf, "", "	")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Server side problem occured while loading conferences info.")
	}

	commiterr := database.CommitConferenceToLocalDB(*newConf)
	if commiterr != nil {
		return commiterr
	}

	return c.SendString(string(newConfJson))
}

func UpdateConference(c *fiber.Ctx) error {
	conference, geterr := database.GetConference(c.Params("id"))
	if geterr := database.HandleGetConferenceError(geterr, c); geterr != nil {
		return geterr
	}

	updatedConf := new(model.Conference)

	if err := c.BodyParser(updatedConf); err != nil {
		return fiber.NewError(
			fiber.StatusBadRequest,
			fmt.Sprintf("Incorrect input for conferrence parameters. Details:\n%v", err))
	}
	updatedConf.ConferenceName = strings.TrimSpace(updatedConf.ConferenceName)

	reqPathParts := strings.Split(c.OriginalURL(), "/")
	var validationErr error = nil
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
		return validationErr
	}

	updatedConf.Id = conference.Id
	updatedConf.RemainingTickets = updatedConf.TotalTickets - database.GetTotalBookings(conference)
	updatedConf.Bookings = conference.Bookings

	updatedConfJson, err := json.MarshalIndent(updatedConf, "", "	")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Server side problem occured while loading conferences info.")
	}

	commiterr := database.CommitConferenceToLocalDB(*updatedConf)
	if commiterr != nil {
		return commiterr
	}

	return c.SendString(string(updatedConfJson))
}

func DeleteConference(c *fiber.Ctx) error {
	conferences, dbreaderr := database.ReadLocalDB()
	if dbreaderr != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "Server side problem occured while reading conferences info from database.")
	}
	confId := c.Params("id")

	for confIndex, conference := range conferences {
		if conference.Id == confId {
			conferences = append(conferences[:confIndex], conferences[confIndex+1:]...)
			database.CommitConferencesToLocalDB(conferences)
			return c.SendStatus(fiber.StatusOK)
		}
	}

	return fiber.NewError(fiber.StatusNotFound, fmt.Sprintf("No conference with id %v to delete.", confId))
}

func validateConferenceInfoInput(conf model.Conference, isNew bool) error {
	nameValidationErr := isValidConferenceName(conf.ConferenceName, isNew)
	totalTicketsValidationErr := isValidConferenceTotalTickets(conf, isNew)
	if nameValidationErr != nil {
		return fiber.NewError(
			fiber.StatusBadRequest,
			fmt.Sprintf("Incorrect input for conferrence name. Details:\n%v", nameValidationErr))
	}
	if totalTicketsValidationErr != nil {
		return fiber.NewError(
			fiber.StatusBadRequest,
			fmt.Sprintf("Incorrect input for conferrence total number of tickets. Details:\n%v", totalTicketsValidationErr))
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
		return false, fiber.NewError(fiber.StatusInternalServerError, "Server side problem occured while reading conferences info from database.")
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
			return fiber.NewError(
				fiber.StatusBadRequest,
				fmt.Sprintf("cannot assign %v as total tickets, %v tickets already booked", conf.TotalTickets, totalBookings))
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
