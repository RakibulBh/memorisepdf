package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var connections = make(map[string]*websocket.Conn)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for development (adjust for production)
		return true
	},
}

func (app *application) startChat(w http.ResponseWriter, r *http.Request) {

	sessionId := uuid.NewString()

	sessionKey := fmt.Sprintf("session:%s", sessionId)

	// cache the store!!!
	app.redis.Set(r.Context(), sessionKey, "{}", 0)

	log.Printf("sessionKey: %s", sessionKey)

	// add the user to a queue
	_, err := app.redis.RPush(r.Context(), "waiting_queue", sessionKey).Result()
	if err != nil {
		http.Error(w, "Failed to add user to queue", http.StatusInternalServerError)
		return
	}

	app.writeJSON(w, http.StatusCreated, "session started dawg", sessionKey)
}

type Message struct {
	RoomId    string `json:"roomId"`
	EventType string `json:"eventType"`
	Content   string `json:"content"`
	To        string `json:"to"`
	From      string `json:"from"`
}

func (app *application) handleChat(w http.ResponseWriter, r *http.Request) {
	sessionKey := chi.URLParam(r, "sessionKey")

	if sessionKey == "" {
		log.Println("Session ID is required")
		http.Error(w, "Session ID is required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade to WebSocket: %v", err)
		http.Error(w, "Failed to upgrade to WebSocket", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	// Clean up the memory
	onClose := func(code int, text string) error {
		delete(connections, sessionKey)
		return nil
	}
	conn.SetCloseHandler(onClose)

	connections[sessionKey] = conn

	for {
		var msg Message
		err = conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("Failed to read message: %v", err)
			http.Error(w, "Failed to read message", http.StatusInternalServerError)
			return
		}

		recipient := msg.To

		recipientConn, ok := connections[recipient]
		if !ok {
			log.Printf("Reciepient %s not found", recipient)
			return
		}

		log.Printf("Sending message to %s", recipient)

		if err := recipientConn.WriteJSON(msg); err != nil {
			http.Error(w, "Failed to write message", http.StatusInternalServerError)
			return
		}
	}
}
