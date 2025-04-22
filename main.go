package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/RakibulBh/relatewme/internal/cache"
	"github.com/RakibulBh/relatewme/internal/env"
	"github.com/google/uuid"
)

func main() {
	if env.GetString("ENV", "development") != "development" {
		fmt.Print("Production environment")
	}

	cfg := config{
		addr: ":" + env.GetString("PORT", "8080"),
		env:  env.GetString("ENV", "development"),
		// db: dbConfig{
		// 	addr:         env.GetString("DB_ADDR", "postgres://admin:adminpassword@localhost:5432/letstalk?sslmode=disable"),
		// 	maxOpenConns: env.GetInt("DB_MAX_OPEN_CONNS", 30),
		// 	maxIdleConns: env.GetInt("DB_MAX_IDLE_CONNS", 30),
		// 	maxIdleTime:  env.GetDuration("DB_MAX_IDLE_TIME", 15*time.Minute).String(),
		// },
		redis: redisConfig{
			addr:     env.GetString("REDIS_PORT", "localhost:6379"),
			password: env.GetString("REDIS_PASSWORD", ""),
			db:       env.GetInt("REDIS_DB", 0),
			protocol: env.GetInt("REDIS_PROTOCOL", 2),
		},
		apiURL: env.GetString("API_URL", "http://localhost:8080"),
	}

	// db, err := db.New(cfg.db.addr, cfg.db.maxOpenConns, cfg.db.maxIdleConns, cfg.db.maxIdleTime)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// defer db.Close()

	// store := store.NewStorage(db)

	// Init redis
	redisClient, err := cache.New(cfg.redis.addr, cfg.redis.password, cfg.redis.db, cfg.redis.protocol)
	if err != nil {
		log.Fatal(err)
	}

	app := &application{
		config: cfg,
		redis:  redisClient,
		// store:  store,
	}

	go app.processWaitingQueue()

	mux := app.serve()
	log.Fatal(app.run(mux))

}

type Room struct {
	Id    string `json:"id"`
	User1 string `json:"user1"`
	User2 string `json:"user2"`
}

func (app *application) processWaitingQueue() {
	for {
		time.Sleep(1 * time.Second)

		if app.redis.LLen(context.Background(), "waiting_queue").Val() < 2 {
			continue
		}

		userSessions, err := app.redis.LPopCount(context.Background(), "waiting_queue", 2).Result()
		if err != nil {
			log.Printf("error in waiting queue: %v", err)
			continue
		}

		// create a room
		roomKey := "room:" + uuid.New().String()

		room := &Room{
			Id:    roomKey,
			User1: userSessions[0],
			User2: userSessions[1],
		}

		user1Payload := Room{
			Id:    roomKey,
			User1: userSessions[0],
			User2: userSessions[1],
		}

		user2Payload := Room{
			Id:    roomKey,
			User1: userSessions[1],
			User2: userSessions[0],
		}

		app.redis.Set(context.Background(), roomKey, room, 0)

		conn1 := connections[userSessions[0]]
		conn2 := connections[userSessions[1]]

		conn1.WriteJSON(map[string]any{"eventType": "room_created", "payload": user1Payload})
		conn2.WriteJSON(map[string]any{"eventType": "room_created", "payload": user2Payload})
	}
}
