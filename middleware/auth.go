package middleware

import (
	"booking-webapp/config"

	"github.com/gofiber/fiber/v2"
	jwtware "github.com/gofiber/jwt/v2"
)

func Authorize() fiber.Handler {
	envval, _ := config.GetSecret("SIGN")

	return jwtware.New(jwtware.Config{
		SigningKey:   []byte(envval),
		ErrorHandler: jwtError,
		ContextKey:   "identity",
	})
}

func jwtError(c *fiber.Ctx, err error) error {
	if err.Error() == "Missing or malformed JWT" {
		return c.Status(fiber.StatusBadRequest).
			JSON(fiber.Map{"status": "error", "message": "Missing or malformed JWT", "data": nil})
	}
	return c.Status(fiber.StatusUnauthorized).
		JSON(fiber.Map{"status": "error", "message": "Invalid or expired JWT", "data": nil})
}
