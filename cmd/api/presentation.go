package main

import (
	"fmt"
	"net/http"
)

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

	fmt.Println(request.PresentationText)

	app.writeJSON(w, http.StatusOK, "Presentation parsed successfully", nil)
}
