package main

import "net/http"

func (app *application) healthCheck(w http.ResponseWriter, r *http.Request) {
	app.writeJSON(w, http.StatusOK, "OK", nil)
}
