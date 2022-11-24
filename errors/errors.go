package errors

import (
	"github.com/gofiber/fiber/v2"
)

func RaiseError(context *fiber.Ctx, status int, message string, data string) error {
	return context.Status(status).JSON(fiber.Map{
		"status":  "error",
		"message": message,
		"data":    data})
}

func RaisePermissionsError(context *fiber.Ctx, data string) error {
	return RaiseError(context, fiber.StatusUnauthorized, "lack of permissions", data)
}

func RaiseInternalServerError(context *fiber.Ctx, data string) error {
	return RaiseError(context, fiber.StatusInternalServerError, "internal error", data)
}

func RaiseBadRequestError(context *fiber.Ctx, data string) error {
	return RaiseError(context, fiber.StatusBadRequest, "bad request", data)
}

func RaiseNotFoundError(context *fiber.Ctx, data string) error {
	return RaiseError(context, fiber.StatusNotFound, "resource not found", data)
}
