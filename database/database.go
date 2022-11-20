package database

import (
	"booking-webapp/config"
	"booking-webapp/model"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ctx = context.TODO()
var UsersCollection *mongo.Collection
var ConferencesCollection *mongo.Collection

func ReadLocalDB() ([]model.Conference, error) {
	conferences := []model.Conference{}

	fileBytes, err := os.ReadFile(config.LOCAL_DB_PATH)
	if os.IsNotExist(err) {
		os.WriteFile(config.LOCAL_DB_PATH, []byte("[]"), 0644)
		fileBytes, _ = os.ReadFile(config.LOCAL_DB_PATH)
	} else if err != nil {
		return nil, err
	}

	err = json.Unmarshal(fileBytes, &conferences)
	if err != nil {
		return nil, err
	}

	return conferences, nil
}

func CommitConferencesToLocalDB(conferences []model.Conference) error {
	conferencesBytes, err := json.MarshalIndent(conferences, "", "	")
	if err != nil {
		return err
	}

	err = os.WriteFile(config.LOCAL_DB_PATH, conferencesBytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

func GetConference(confId string) (model.Conference, error) {
	conferences, readerr := ReadLocalDB()
	if readerr != nil {
		return model.Conference{}, errors.New("server side problem occured while reading conferences info from database")
	}

	for _, conference := range conferences {
		if conference.Id == confId {
			return conference, nil
		}
	}

	return model.Conference{}, fmt.Errorf("no conference with id %v in database", confId)
}

func HandleGetConferenceError(geterr error, c *fiber.Ctx) error {
	if geterr != nil {
		if strings.HasPrefix(fmt.Sprintf("%v", geterr), "no conference") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"status":  "error",
				"message": "Error on getting info from database",
				"data":    geterr})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"status":  "error",
			"message": "Error on getting info from database",
			"data":    geterr})
	}
	return nil
}

func CommitConferenceToLocalDB(conference model.Conference) error {
	conferences, err := ReadLocalDB()
	if err != nil {
		return err
	}

	for confIndex, conf := range conferences {
		if conf.Id == conference.Id {
			conferences[confIndex] = conference
			commiterr := CommitConferencesToLocalDB(conferences)
			if commiterr != nil {
				return commiterr
			}
			return nil
		}
	}

	conferences = append(conferences, conference)
	commiterr := CommitConferencesToLocalDB(conferences)
	if commiterr != nil {
		return commiterr
	}

	return nil
}

func GetTotalBookings(conf model.Conference) uint {
	var totalBookings uint = 0
	for _, booking := range conf.Bookings {
		if !booking.IsCanceled {
			totalBookings += booking.TicketsBooked
		}
	}
	return totalBookings
}

func DBInit(collectionName string) (*mongo.Collection, error) {
	connString, err := config.GetSecret("MONGODB_CONNSTRING")
	if err != nil {
		log.Fatal("cannot find connection string for DB in the environment")
	}

	clientOptions := options.Client().ApplyURI(connString)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to the db: %v", err)
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("db is not available: %v", err)
	}

	return client.Database("booking-service").Collection(collectionName), nil
}

func GetUserData(userLogin string) (model.UserData, error) {
	var user model.UserData
	cur, err := UsersCollection.Find(ctx, bson.D{primitive.E{Key: "login", Value: userLogin}})
	if err != nil {
		return model.UserData{}, fmt.Errorf("server side problem occured while reading user data from database: %v", err)
	}

	for cur.Next(ctx) {
		err := cur.Decode(&user)
		if err != nil {
			return model.UserData{}, fmt.Errorf("server side problem occured while reading user data from database: %v", err)
		}
	}

	if err := cur.Err(); err != nil {
		return model.UserData{}, fmt.Errorf("server side problem occured while reading user data from database: %v", err)
	}

	cur.Close(ctx)

	return user, nil
}
