package main

import (
	"fmt"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func (app *application) GetLogs(w http.ResponseWriter, r *http.Request) {
	collection := app.logger.Client.Database("logs").Collection("success_logs")

	// Set options to sort by createdAt in descending order (newest first)
	// Also limit to 50 most recent logs
	findOptions := options.Find().
		SetSort(bson.M{"createdAt": -1}).
		SetLimit(50)

	cursor, err := collection.Find(r.Context(), bson.M{}, findOptions)
	if err != nil {
		app.errorJSON(w, err)
		return
	}

	var logs []LogEntry
	if err := cursor.All(r.Context(), &logs); err != nil {
		app.errorJSON(w, err)
		return
	}

	// Debug output
	for _, result := range logs {
		res, _ := bson.MarshalExtJSON(result, false, false)
		fmt.Println(string(res))
	}

	app.writeJSON(w, http.StatusOK, "logs", logs)
}
