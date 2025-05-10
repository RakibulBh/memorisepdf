package logger

import (
	"context"
	"time"

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

func (l Logger) LogSuccessfulOutput(ctx context.Context, message string, fileName string, fileSize string, operation string, fileFormat string, ipAddress string) error {
	collection := l.Client.Database("logs").Collection("success_logs")

	doc := bson.M{
		"fileName":   fileName,
		"fileSize":   fileSize,
		"fileFormat": fileFormat,
		"message":    message,
		"operation":  operation,
		"ipAddress":  ipAddress,
		"createdAt":  time.Now(),
	}

	_, err := collection.InsertOne(ctx, doc)
	return err
}

func (l Logger) LogFailedOutput(ctx context.Context, message string, fileName string, fileSize string, operation string, fileFormat string, ipAddress string) error {
	collection := l.Client.Database("logs").Collection("failed_logs")

	doc := bson.M{
		"fileName":   fileName,
		"fileSize":   fileSize,
		"fileFormat": fileFormat,
		"message":    message,
		"operation":  operation,
		"ipAddress":  ipAddress,
		"createdAt":  time.Now(),
	}

	_, err := collection.InsertOne(ctx, doc)
	return err
}
