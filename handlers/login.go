package handlers

import (
	"booking-webapp/config"
	"booking-webapp/database"
	"booking-webapp/model"
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
		Login    string `json:"login" bson:"login,omitempty"`
		Password string `json:"password" bson:"password,omitempty"`
	}

	var creds = new(Credentials)

	isAnonymousCall := false

	err := c.BodyParser(&creds)
	if err == fiber.ErrUnprocessableEntity {
		isAnonymousCall = true
	} else if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Error on login request when parse credentials",
			"data":    err})
	}

	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["exp"] = time.Now().Add(time.Hour * 8).Unix()

	if isAnonymousCall {
		claims["username"] = "anonymous"
		claims["role"] = "anonymous"
	} else {
		user, autherr := authenticate(creds.Login, creds.Password)
		if autherr != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"status":  "error",
				"message": "Error on login request when comparing user data",
				"data":    fmt.Sprint(autherr)})
		}

		claims["username"] = user.Login
		claims["role"] = user.Role
	}

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

func authenticate(login string, pass string) (model.UserData, error) {
	user, geterr := database.GetUserData(login)
	if geterr != nil {
		return model.UserData{}, fmt.Errorf("%v", geterr)
	}

	if !isPasswordHashCorrect(user.HashedPassword, pass) {
		return model.UserData{}, fmt.Errorf("invalid password")
	}

	return user, nil
}
