package config

import "os"

type Config struct {
	Addr        string
	BaseURL     string
	DatabaseURL string
	AdminToken  string
	APIToken    string
}

func Load() Config {
	addr := os.Getenv("RELAY_ADDR")
	if addr == "" {
		addr = ":8080"
	}

	databaseURL := os.Getenv("RELAY_DATABASE_URL")
	adminToken := firstNonEmpty(os.Getenv("RELAY_ADMIN_TOKEN"), os.Getenv("RELAY_API_TOKEN"))
	apiToken := os.Getenv("RELAY_API_TOKEN")
	baseURL := os.Getenv("RELAY_BASE_URL")

	return Config{
		Addr:        addr,
		BaseURL:     baseURL,
		DatabaseURL: databaseURL,
		AdminToken:  adminToken,
		APIToken:    apiToken,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
