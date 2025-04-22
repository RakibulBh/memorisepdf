package main

import (
	"fmt"
	"html"
	"net/http"
	"strings"

	"google.golang.org/genai"
)

// Sanitize input to prevent prompt injection
func sanitizeInput(input string) string {
	// Remove any attempt to use markdown code blocks which could be used to escape context
	input = strings.ReplaceAll(input, "```", "")

	// HTML encode to prevent special character usage
	input = html.EscapeString(input)

	// Remove any potential directive-like patterns
	input = strings.ReplaceAll(input, "system:", "")
	input = strings.ReplaceAll(input, "assistant:", "")
	input = strings.ReplaceAll(input, "user:", "")
	input = strings.ReplaceAll(input, "instructions:", "")

	return input
}

func generatePresentationPrompt(useCase string, presentationText string) string {
	// Sanitize the user input
	safeText := sanitizeInput(presentationText)

	switch useCase {
	case "presentation":
		// Create clear boundaries between system instructions and user content
		return presentationPrompt + "\n\n### USER CONTENT START ###\n" + safeText + "\n### USER CONTENT END ###\n\nAnalyze ONLY the content between the USER CONTENT markers."
	case "quiz":
		return ""
	default:
		return ""
	}
}

type presentationRequest struct {
	PresentationText string `json:"presentation_text"`
}

func (app *application) parsePresentation(w http.ResponseWriter, r *http.Request) {
	var request presentationRequest
	err := app.readJSON(w, r, &request)
	if err != nil {
		app.badRequestError(w, err.Error())
		return
	}

	// Validate input
	if strings.TrimSpace(request.PresentationText) == "" {
		app.badRequestError(w, "presentation text cannot be empty")
		return
	}

	// Limit input size to prevent excessive token usage
	if len(request.PresentationText) > 50000 {
		app.badRequestError(w, "presentation text exceeds maximum length")
		return
	}

	fmt.Println(request.PresentationText)

	ctx := r.Context()

	result, err := app.llm.Models.GenerateContent(ctx, app.config.geimini.model, genai.Text(generatePresentationPrompt("presentation", request.PresentationText)), &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		MaxOutputTokens:  4096,
		Temperature:      genai.Ptr[float32](0.5),
	})
	if err != nil {
		app.internalServerError(w, err.Error())
		return
	}

	app.writeJSON(w, http.StatusOK, "Presentation parsed successfully", result.Text())
}
