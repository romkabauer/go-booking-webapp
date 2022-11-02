package handlers

import (
	"booking-webapp/config"
	"booking-webapp/database"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt"
	"golang.org/x/crypto/bcrypt"
)

func isPasswordHashCorrect(dbHash, pass string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(dbHash), []byte(pass))
	return err == nil
}

func Login(c *fiber.Ctx) error {
	type Credentials struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}

	var creds = new(Credentials)

	if err := c.BodyParser(&creds); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Error on login request when parse credentials",
			"data":    err})
	}

	user, geterr := database.GetUserData(creds.Login)
	if geterr != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Error on login request when comparing user data",
			"data":    fmt.Sprintf("%v", geterr)})
	}

	if !isPasswordHashCorrect(user.HashedPassword, creds.Password) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"status":  "error",
			"message": "Invalid password",
			"data":    nil})
	}

	token := jwt.New(jwt.SigningMethodHS256)

	claims := token.Claims.(jwt.MapClaims)
	claims["username"] = user.Login
	claims["exp"] = time.Now().Add(time.Hour * 8).Unix()
	claims["role"] = user.Role

	sign, enverr := config.GetSecret("SIGN")
	if enverr != nil {
		log.Print(enverr)
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	t, err := token.SignedString([]byte(sign))
	if err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	return c.JSON(fiber.Map{"status": "success", "message": "Success login", "data": t})
}
