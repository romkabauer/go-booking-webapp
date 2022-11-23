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
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetConferences(c *fiber.Ctx) error {
	conferences, dberr := database.GetConferences(isAdminRole(c), "")
	if dberr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while database call",
			"data":    fmt.Sprint(dberr)})
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
	conferences, dberr := database.GetConferences(isAdminRole(c), "_id", c.Params("id"))
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

	conferenceJson, err := json.MarshalIndent(conferences[0], "", "	")
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

	newConf.Id = primitive.NewObjectID()
	newConf.RemainingTickets = newConf.TotalTickets
	newConf.Bookings = []model.Booking{}

	newConfJson, err := json.MarshalIndent(newConf, "", "	")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while sending conference info to client",
			"data":    err})
	}

	commiterr := database.WriteToCollection(*newConf, database.ConferencesCollection)
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

	conferences, dberr := database.GetConferences(isAdminRole(c), "_id", c.Params("id"))
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

	updatedConf := new(model.Conference)

	if err := c.BodyParser(updatedConf); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "incorrect input for conferrence parameters",
			"data":    fmt.Sprint(err)})
	}
	updatedConf.Id = conferences[0].Id
	updatedConf.ConferenceName = strings.TrimSpace(updatedConf.ConferenceName)

	reqPathParts := strings.Split(c.OriginalURL(), "/")
	var validationErr error = nil
	// adjust validation according to the request path, e.g. do only name validation if name update occure
	if reqPathParts[len(reqPathParts)-1] == "name" {
		updatedConf.TotalTickets = conferences[0].TotalTickets
		validationErr = isValidConferenceName(updatedConf.ConferenceName, false)
	} else if reqPathParts[len(reqPathParts)-1] == "tickets" {
		updatedConf.ConferenceName = conferences[0].ConferenceName
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

	totalBookings, err := database.GetTotalBookings(updatedConf.Id)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "database error occured",
			"data":    fmt.Sprint(err)})
	}

	updatedConf.RemainingTickets = updatedConf.TotalTickets - totalBookings
	updatedConf.Bookings = conferences[0].Bookings

	updatedConfJson, err := json.MarshalIndent(updatedConf, "", "	")
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "server side problem occured while sending conference info to client",
			"data":    err})
	}

	commiterr := database.UpdateCollectionItem(updatedConf.Id, updatedConf, database.ConferencesCollection)
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
	deleteErr := database.DeleteFromCollection(c.Params("id"), database.ConferencesCollection)
	if deleteErr != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"status":  "error",
			"message": "unsuccessful deletion",
			"data":    fmt.Sprintf("failed to delete: %v", deleteErr)})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "conference deleted",
		"data":    fmt.Sprintf("conference with id %v was deleted", c.Params("id"))})
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
	}

	nameExists, err := database.IfConferenceNameAlreadyExist(name)
	if err != nil {
		return err
	}
	if nameExists {
		return errors.New("conference name already exist")
	}

	return nil
}

func isValidConferenceTotalTickets(conf model.Conference, isNew bool) error {
	if conf.TotalTickets == 0 {
		return errors.New("conference cannot have zero tickets for distribution")
	}
	if !isNew {
		totalBookings, err := database.GetTotalBookings(conf.Id)
		if err != nil {
			return fmt.Errorf("database error occured: %v", err)
		}

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
