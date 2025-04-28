package logger

import (
	"context"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Logger struct {
	Client *mongo.Client
}

func New(mongUri string) (*mongo.Client, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(mongUri))
	if err != nil {
		return nil, err
	}

	return client, err
}

func (l Logger) LogSuccessfulOutput(ctx context.Context, message string, fileName string, fileSize string, operation string, fileFormat string) error {
	collection := l.Client.Database("logs").Collection("success_logs")

	_, err := collection.InsertOne(ctx, bson.D{
		{Key: "fileName", Value: fileName},
		{Key: "fileSize", Value: fileSize},
		{Key: "fileFormat", Value: fileFormat},
		{Key: "message", Value: message},
		{Key: "operation", Value: operation},
	})

	return err
}
