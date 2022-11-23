package handlers

import (
	"booking-webapp/database"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GetHello(c *fiber.Ctx) error {
	objId, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "db error",
			"data":    fmt.Sprint(err)})
	}

	conferences, err := database.GetTotalBookings(objId)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "some db error",
			"data":    fmt.Sprint(err)})
	}

	return c.SendString(fmt.Sprintf("%v\n", conferences))
}
