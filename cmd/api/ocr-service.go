package main

import (
	"context"
	"fmt"
	"mime/multipart"

	"github.com/google/go-tika/tika"
)

func (app *application) parseFile(file *multipart.File) (string, error) {
	client := tika.NewClient(nil, "https://apache-tike.fly.dev")
	text, err := client.Parse(context.Background(), *file)
	if err != nil {
		fmt.Println("Error parsing file:", err)
		return "", err
	}

	return text, nil
}
