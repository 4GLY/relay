package config

import "os"

type Config struct {
	Addr        string
	DatabaseURL string
	APIToken    string
}

func Load() Config {
	addr := os.Getenv("RELAY_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	databaseURL := os.Getenv("RELAY_DATABASE_URL")
	apiToken := os.Getenv("RELAY_API_TOKEN")

	return Config{
		Addr:        addr,
		DatabaseURL: databaseURL,
		APIToken:    apiToken,
	}
}
