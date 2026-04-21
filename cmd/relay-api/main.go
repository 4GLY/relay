package main

import (
	"log"

	"relay/internal/api"
	"relay/internal/config"
)

func main() {
	cfg := config.Load()
	if err := api.ListenAndServe(cfg); err != nil {
		log.Fatal(err)
	}
}
