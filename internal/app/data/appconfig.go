package data

import (
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AppConfig struct {
	Id         primitive.ObjectID `bson:"_id"`
	ConfigBody string             `bson:"config_body"`
	Previous   primitive.ObjectID `bson:"previous"`
}

func newAppConfig(configBody string) *AppConfig {
	return &AppConfig{
		Id:         primitive.NewObjectID(),
		ConfigBody: configBody,
	}
}

func (controller *DBController) AddConfig(configBody string) error {
	_, err := controller.collection.InsertOne(controller.ctx, newAppConfig(configBody))
	return err
}

func (controller *DBController) FindConfig() (string, error) {
	var foundConfig AppConfig
	opt := options.FindOne().SetSort(bson.M{"$natural": -1})
	err := controller.collection.FindOne(controller.ctx, bson.M{}, opt).Decode(&foundConfig)
	if err != nil && err == mongo.ErrNoDocuments {
		return "", err
	}
	return foundConfig.ConfigBody, nil
}

func (controller *DBController) UpdateConfig(configBody string) error {
	var foundConfig AppConfig
	opt := options.FindOne().SetSort(bson.M{"$natural": -1})
	err := controller.collection.FindOne(controller.ctx, bson.M{}, opt).Decode(&foundConfig)
	if err != nil {
		return err
	}
	newConfig := AppConfig{
		Id:         primitive.NewObjectID(),
		ConfigBody: configBody,
		Previous:   foundConfig.Id,
	}
	_, err = controller.collection.InsertOne(controller.ctx, newConfig)
	if err != nil {
		return err
	}
	return nil
}

func (controller *DBController) DeleteConfig() error {
	if err := controller.collection.Drop(controller.ctx); err != nil {
		return err
	}
	return nil
}
