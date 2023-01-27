package data

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type User struct {
	Id            primitive.ObjectID `bson:"_id"`
	Username      string             `bson:"username"`
	AcceptedToken string             `bson:"accepted_token"`
}

func newUser(name string, atoken string) *User {
	return &User{
		Id:            primitive.NewObjectID(),
		Username:      name,
		AcceptedToken: atoken,
	}
}

func (controller *DBController) AddUser(usrn string, accToken string) error {
	_, err := controller.collection.InsertOne(controller.ctx, newUser(usrn, accToken))
	return err
}

func (controller *DBController) FindUserByName(usrn string) (*User, error) {
	var findedUser User
	err := controller.collection.FindOne(context.Background(), bson.D{{"username", usrn}}).Decode(&findedUser)
	if err != nil && err == mongo.ErrNoDocuments {
		return &User{}, err
	}
	return &findedUser, nil
}

func (controller *DBController) FindUserByToken(accToken string) (*User, error) {
	var foundUser User
	err := controller.collection.FindOne(context.Background(), bson.D{{"accepted_token", accToken}}).Decode(&foundUser)
	if err != nil && err == mongo.ErrNoDocuments {
		return &User{}, err
	}
	return &foundUser, nil
}

func (controller *DBController) GetUserToken(username string) (string, error) {
	var foundUser User
	err := controller.collection.FindOne(context.Background(), bson.D{{"username", username}}).Decode(&foundUser)
	if err != nil && err == mongo.ErrNoDocuments {
		return "", err
	}
	return foundUser.AcceptedToken, nil
}
