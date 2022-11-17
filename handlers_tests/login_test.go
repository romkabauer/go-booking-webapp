package handlers

import (
	"booking-webapp/router"
	"bytes"
	"io"
	"strings"

	"net/http"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

type Test struct {
	description   string
	route         string
	bodyinput     []byte
	expectedError bool
	expectedCode  int
	expectedBody  string
}

func TestLogin(t *testing.T) {
	tests := []Test{
		{
			description:   "login anonymous",
			route:         "/login",
			bodyinput:     nil,
			expectedError: false,
			expectedCode:  200,
			expectedBody:  "",
		},
		{
			description:   "user login",
			route:         "/login",
			bodyinput:     []byte("{\"login\":\"fake_admin\",\"password\":\"admin\"}"),
			expectedError: false,
			expectedCode:  200,
			expectedBody:  "",
		}}

	app := fiber.New()
	router.SetupRoutes(app)

	for _, test := range tests {
		req, _ := http.NewRequest(
			"POST",
			test.route,
			bytes.NewBuffer(test.bodyinput))

		res, _ := app.Test(req, -1)

		body := new(strings.Builder)
		_, err := io.Copy(body, res.Body)
		if err != nil {
			assert.Fail(t, "Invalid test, error occured while body parsing")
		}

		assert.Equalf(t, test.expectedCode, res.StatusCode, test.description)
	}

}
