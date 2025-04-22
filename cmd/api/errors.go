package main

import "net/http"

func (a *application) internalServerError(w http.ResponseWriter, err string) {
	a.writeJSON(w, http.StatusInternalServerError, err, nil)
}

func (a *application) notFoundError(w http.ResponseWriter, err string) {
	a.writeJSON(w, http.StatusNotFound, err, nil)
}

func (a *application) badRequestError(w http.ResponseWriter, err string) {
	a.writeJSON(w, http.StatusBadRequest, err, nil)
}
