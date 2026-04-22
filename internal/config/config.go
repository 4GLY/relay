package config

import "os"

type Config struct {
	Addr        string
	BaseURL     string
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
	baseURL := os.Getenv("RELAY_BASE_URL")

	return Config{
		Addr:        addr,
		BaseURL:     baseURL,
		DatabaseURL: databaseURL,
		APIToken:    apiToken,
	}
}
