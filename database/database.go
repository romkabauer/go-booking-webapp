package database

import (
	"booking-webapp/config"
	"booking-webapp/model"
	"context"
	"fmt"
	"log"
	"reflect"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var mongoCtx = context.TODO()
var UsersCollection *mongo.Collection
var ConferencesCollection *mongo.Collection

func DBInit(collectionName string) (*mongo.Collection, error) {
	connString, err := config.GetSecret("MONGODB_CONNSTRING")
	if err != nil {
		log.Fatal("cannot find connection string for DB in the environment")
	}

	clientOptions := options.Client().ApplyURI(connString)
	client, err := mongo.Connect(mongoCtx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to the db: %v", err)
	}

	err = client.Ping(mongoCtx, nil)
	if err != nil {
		return nil, fmt.Errorf("db is not available: %v", err)
	}

	return client.Database("booking-service").Collection(collectionName), nil
}

func GetUserData(userLogin string) (model.UserData, error) {
	var user model.UserData
	cur, err := UsersCollection.Find(mongoCtx, bson.D{primitive.E{Key: "login", Value: userLogin}})
	if err != nil {
		return model.UserData{}, fmt.Errorf("server side problem occured while reading user data from database: %v", err)
	}

	for cur.Next(mongoCtx) {
		err := cur.Decode(&user)
		if err != nil {
			return model.UserData{}, fmt.Errorf("server side problem occured while reading user data from database: %v", err)
		}
	}

	if err := cur.Err(); err != nil {
		return model.UserData{}, fmt.Errorf("server side problem occured while reading user data from database: %v", err)
	}

	cur.Close(mongoCtx)

	return user, nil
}

func GetConferences(isPriveleged bool, filterField string, params ...string) ([]model.Conference, error) {
	var conferences []model.Conference

	var filter primitive.D = bson.D{{}}
	if len(params) != 0 {
		filter = bson.D{
			{
				Key: filterField, Value: bson.D{
					{Key: "$in", Value: toBsonArray(params, true)},
				},
			},
		}
	}

	var projection primitive.D = bson.D{
		{Key: "_id", Value: 1},
		{Key: "conference_name", Value: 1},
		{Key: "total_tickets", Value: 1},
		{Key: "remaining_tickets", Value: 1},
	}
	if isPriveleged {
		projection = append(projection, primitive.E{Key: "bookings", Value: 1})
	}

	cur, err := ConferencesCollection.Find(mongoCtx, filter, options.Find().SetProjection(projection))
	if err != nil {
		return conferences, fmt.Errorf("server side problem occured while reading user data from database: %v", err)
	}

	cur.All(mongoCtx, &conferences)

	if err := cur.Err(); err != nil {
		return conferences, fmt.Errorf("server side problem occured while reading user data from database: %v", err)
	}

	return conferences, nil
}

func WriteToCollection(obj any, collection *mongo.Collection) error {
	newObj, err := toBsonObj(obj)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	_, writeErr := ConferencesCollection.InsertOne(mongoCtx, newObj)
	if writeErr != nil {
		return fmt.Errorf("%v", writeErr)
	}

	return nil
}

func UpdateCollectionItem(objId primitive.ObjectID, objData any, collection *mongo.Collection) error {
	newObj, err := toBsonObj(objData)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	_, writeErr := ConferencesCollection.UpdateOne(
		mongoCtx,
		bson.D{
			{Key: "_id", Value: objId},
		},
		bson.D{
			{Key: "$set", Value: *newObj},
		},
	)
	if writeErr != nil {
		return fmt.Errorf("%v", writeErr)
	}

	return nil
}

func InsertBooking(conference model.Conference, booking model.Booking, index ...uint) error {
	conference.RemainingTickets = conference.RemainingTickets - booking.TicketsBooked
	if len(index) == 0 {
		conference.Bookings = append(conference.Bookings, booking)
	} else if booking.IsCanceled {
		conference.RemainingTickets += booking.TicketsBooked
		booking.UpdatedAt = time.Now().Format(time.RFC3339)
		conference.Bookings[index[0]] = booking
	} else {
		deltaTickets := int(booking.TicketsBooked) - int(conference.Bookings[index[0]].TicketsBooked)
		conference.RemainingTickets = uint(int(conference.RemainingTickets) - deltaTickets)
		booking.UpdatedAt = time.Now().Format(time.RFC3339)
		conference.Bookings[index[0]] = booking
	}

	updateErr := UpdateCollectionItem(
		conference.Id,
		conference,
		ConferencesCollection)

	return updateErr
}

func DeleteFromCollection(objId string, collection *mongo.Collection) error {
	result, deleteErr := ConferencesCollection.DeleteOne(
		mongoCtx,
		bson.D{
			{Key: "_id", Value: toBsonArray([]string{objId}, true)[0]},
		})
	if deleteErr != nil {
		return fmt.Errorf("%v", deleteErr)
	}
	if result.DeletedCount == 0 {
		return fmt.Errorf("nothing was deleted")
	}
	return nil
}

func GetTotalBookings(confId primitive.ObjectID) (uint, error) {
	matchStage := bson.D{
		{
			Key: "$match", Value: bson.D{
				{Key: "_id", Value: confId},
			},
		},
	}
	unwindStage := bson.D{
		{Key: "$unwind", Value: "$bookings"},
	}
	filterCanceledStage := bson.D{
		{
			Key: "$match", Value: bson.D{
				{Key: "bookings.is_canceled", Value: false},
			},
		},
	}
	groupStage := bson.D{
		{
			Key: "$group", Value: bson.D{
				{Key: "_id", Value: "$conference_name"},
				{
					Key: "total_bookings", Value: bson.D{
						{Key: "$sum", Value: "$bookings.tickets_booked"},
					},
				},
			},
		},
	}

	cur, err := ConferencesCollection.Aggregate(mongoCtx,
		mongo.Pipeline{
			matchStage,
			unwindStage,
			filterCanceledStage,
			groupStage,
		})
	if err != nil {
		return 0, fmt.Errorf("error while database call: %v", err)
	}

	var results []bson.M
	if err = cur.All(mongoCtx, &results); err != nil {
		return 0, fmt.Errorf("error while database call: %v", err)
	}

	if len(results) == 0 {
		log.Printf("nothing to aggregate for %v\n", confId)
		return 0, nil
	}

	switch results[0]["total_bookings"].(type) {
	case int8:
		return uint(results[0]["total_bookings"].(int8)), nil
	case int16:
		return uint(results[0]["total_bookings"].(int16)), nil
	case int32:
		return uint(results[0]["total_bookings"].(int32)), nil
	case int64:
		return uint(results[0]["total_bookings"].(int64)), nil
	default:
		return 0, fmt.Errorf("aggregation failed trying to cast %v to int, type of value is %v",
			results[0]["total_bookings"],
			reflect.TypeOf(results[0]["total_bookings"]).String())
	}

}

func IfConferenceNameAlreadyExist(name string) (bool, error) {
	sameNameConfs, err := GetConferences(false, "conference_name", name)
	if err != nil {
		return true, fmt.Errorf("%v", err)
	}

	if len(sameNameConfs) != 0 {
		return true, fmt.Errorf("conference with the same name already exists")
	}

	return false, nil
}

func toBsonObj(objData any) (bsonObj *bson.D, err error) {
	bsonData, err := bson.Marshal(objData)
	if err != nil {
		return
	}

	err = bson.Unmarshal(bsonData, &bsonObj)
	if err != nil {
		return
	}

	return
}

func toBsonArray(stringArray []string, areIds bool) primitive.A {
	var result primitive.A
	for _, str := range stringArray {
		objId, err := primitive.ObjectIDFromHex(str)
		if areIds && (err == nil) {
			result = append(result, objId)
		} else {
			result = append(result, str)
		}
	}
	return result
}
