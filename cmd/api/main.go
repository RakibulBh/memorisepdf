package main

import (
	"fmt"
	"log"

	"github.com/RakibulBh/relatewme/internal/cache"
	"github.com/RakibulBh/relatewme/internal/env"
)

func main() {
	if env.GetString("ENV", "development") != "development" {
		fmt.Print("Production environment")
	}

	cfg := config{
		addr: ":" + env.GetString("PORT", "8080"),
		env:  env.GetString("ENV", "development"),
		redis: redisConfig{
			addr:     env.GetString("REDIS_PORT", "localhost:6379"),
			password: env.GetString("REDIS_PASSWORD", ""),
			db:       env.GetInt("REDIS_DB", 0),
			protocol: env.GetInt("REDIS_PROTOCOL", 2),
		},
		apiURL: env.GetString("API_URL", "http://localhost:8080"),
	}

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

	mux := app.serve()
	log.Fatal(app.run(mux))

}
