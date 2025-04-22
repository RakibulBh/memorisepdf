package llm

import (
	"context"
	"log"

	"google.golang.org/genai"
)

func New(model string, apiKey string) *genai.Client {
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey: apiKey,
	})

	if err != nil {
		log.Fatal(err)
	}

	return client
}
