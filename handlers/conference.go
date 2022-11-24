package handlers

import (
	"booking-webapp/database"
	"booking-webapp/errors"
	"booking-webapp/model"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v4"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetConferences(c *fiber.Ctx) error {
	conferences, dbErr := database.GetConferences(isAdminRole(c), "")
	if dbErr != nil {
		return errors.RaiseInternalServerError(c, fmt.Sprintf("database error: %v", dbErr))
	}

	conferencesJson, jsonErr := json.MarshalIndent(conferences, "", "	")
	if jsonErr != nil {
		return errors.RaiseInternalServerError(c, fmt.Sprintf("json serialization error: %v", jsonErr))
	}

	return c.SendString(string(conferencesJson))
}

func GetConference(c *fiber.Ctx) error {
	conferences, dbErr := database.GetConferences(isAdminRole(c), "_id", c.Params("confId"))
	if dbErr != nil {
		return errors.RaiseInternalServerError(c, fmt.Sprintf("database error: %v", dbErr))
	}
	if len(conferences) == 0 {
		return errors.RaiseNotFoundError(c, fmt.Sprintf("conference %v not found", c.Params("confId")))
	}

	conferenceJson, jsonErr := json.MarshalIndent(conferences[0], "", "	")
	if jsonErr != nil {
		return errors.RaiseInternalServerError(c, fmt.Sprintf("json serialization error: %v", jsonErr))
	}

	return c.SendString(string(conferenceJson))
}

func CreateNewConference(c *fiber.Ctx) error {
	if !isAdminRole(c) {
		return errors.RaisePermissionsError(c, "only admin can perform this operation")
	}

	newConf := new(model.Conference)
	if jsonErr := c.BodyParser(newConf); jsonErr != nil {
		return errors.RaiseBadRequestError(c, fmt.Sprintf("unacceptable conference parameters: %v", jsonErr))
	}
	newConf.Id = primitive.NewObjectID()
	newConf.ConferenceName = strings.TrimSpace(newConf.ConferenceName)
	newConf.RemainingTickets = newConf.TotalTickets
	newConf.Bookings = []model.Booking{}

	validationErr := validateConferenceInfoInput(*newConf, true)
	if validationErr != nil {
		return errors.RaiseBadRequestError(c,
			fmt.Sprintf("incorrect input for conference parameters: %v", validationErr))
	}

	writeErr := database.WriteToCollection(*newConf, database.ConferencesCollection)
	if writeErr != nil {
		return errors.RaiseInternalServerError(c, fmt.Sprintf("database error: %v", writeErr))
	}

	newConfJson, jsonErr := json.MarshalIndent(newConf, "", "	")
	if jsonErr != nil {
		return errors.RaiseInternalServerError(c, fmt.Sprintf("json serialization error: %v", jsonErr))
	}

	return c.SendString(string(newConfJson))
}

func UpdateConference(c *fiber.Ctx) error {
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

	updatedConf := new(model.Conference)
	if jsonErr := c.BodyParser(updatedConf); jsonErr != nil {
		return errors.RaiseBadRequestError(c, fmt.Sprintf("unacceptable conference parameters: %v", jsonErr))
	}
	updatedConf.Id = conferences[0].Id
	updatedConf.ConferenceName = strings.TrimSpace(updatedConf.ConferenceName)
	updatedConf.Bookings = conferences[0].Bookings

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
		return errors.RaiseBadRequestError(c,
			fmt.Sprintf("incorrect input for conference parameters: %v", validationErr))
	}

	totalBookings, dbErr := database.GetTotalBookings(updatedConf.Id)
	if dbErr != nil {
		return errors.RaiseInternalServerError(c, fmt.Sprintf("database error: %v", dbErr))
	}
	updatedConf.RemainingTickets = updatedConf.TotalTickets - totalBookings

	updateErr := database.UpdateCollectionItem(updatedConf.Id, updatedConf, database.ConferencesCollection)
	if updateErr != nil {
		return errors.RaiseInternalServerError(c, fmt.Sprintf("database error: %v", updateErr))
	}

	updatedConfJson, jsonErr := json.MarshalIndent(updatedConf, "", "	")
	if jsonErr != nil {
		return errors.RaiseInternalServerError(c, fmt.Sprintf("json serialization error: %v", jsonErr))
	}

	return c.SendString(string(updatedConfJson))
}

func DeleteConference(c *fiber.Ctx) error {
	if !isAdminRole(c) {
		return errors.RaisePermissionsError(c, "only admin can perform this operation")
	}
	deleteErr := database.DeleteFromCollection(c.Params("id"), database.ConferencesCollection)
	if deleteErr != nil {
		return errors.RaiseBadRequestError(c, fmt.Sprintf("failed to delete: %v", deleteErr))
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "success",
		"message": "entity deleted",
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
		return fmt.Errorf("conference name is too short")
	}

	nameExists, err := database.IfConferenceNameAlreadyExist(name)
	if err != nil {
		return err
	}
	if nameExists {
		return fmt.Errorf("conference name already exist")
	}

	return nil
}

func isValidConferenceTotalTickets(conf model.Conference, isNew bool) error {
	if conf.TotalTickets == 0 {
		return fmt.Errorf("conference cannot have zero tickets for distribution")
	}
	if !isNew {
		totalBookings, err := database.GetTotalBookings(conf.Id)
		if err != nil {
			return fmt.Errorf("database error occured: %v", err)
		}

		if conf.TotalTickets < totalBookings {
			return fmt.Errorf("cannot assign %v as total tickets, %v tickets already booked",
				conf.TotalTickets,
				totalBookings)
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
