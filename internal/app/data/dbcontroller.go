package data

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type DBController struct {
	collection *mongo.Collection
	ctx        context.Context
}

func NewController(MDBCon string, dataBase string, coll string) (*DBController, error) {
	clientOptions := options.Client().ApplyURI(MDBCon)
	ctx := context.TODO()
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return &DBController{}, err
	}
	err = client.Ping(ctx, nil)
	if err != nil {
		return &DBController{}, err
	}
	collection := client.Database(dataBase).Collection(coll)
	return &DBController{
		collection: collection,
		ctx:        ctx,
	}, nil
}
