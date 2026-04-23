package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr                 string
	BaseURL              string
	DatabaseURL          string
	AdminToken           string
	APIToken             string
	CuratorWorkerID      string
	CuratorProvider      string
	CuratorModel         string
	CuratorBatchSize     int
	CuratorPollInterval  time.Duration
	CuratorLeaseDuration time.Duration
	CuratorRetryBackoff  time.Duration
	CuratorMaxAttempts   int
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
	workerID := os.Getenv("RELAY_CURATOR_WORKER_ID")
	if workerID == "" {
		workerID = "relay-curator"
	}
	provider := os.Getenv("RELAY_CURATOR_PROVIDER")
	if provider == "" {
		provider = "rule-based"
	}

	return Config{
		Addr:                 addr,
		BaseURL:              baseURL,
		DatabaseURL:          databaseURL,
		AdminToken:           adminToken,
		APIToken:             apiToken,
		CuratorWorkerID:      workerID,
		CuratorProvider:      provider,
		CuratorModel:         os.Getenv("RELAY_CURATOR_MODEL"),
		CuratorBatchSize:     envInt("RELAY_CURATOR_BATCH_SIZE", 5),
		CuratorPollInterval:  envDuration("RELAY_CURATOR_POLL_INTERVAL", 5*time.Second),
		CuratorLeaseDuration: envDuration("RELAY_CURATOR_LEASE_DURATION", 30*time.Second),
		CuratorRetryBackoff:  envDuration("RELAY_CURATOR_RETRY_BACKOFF", 30*time.Second),
		CuratorMaxAttempts:   envInt("RELAY_CURATOR_MAX_ATTEMPTS", 5),
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

func envInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func envDuration(name string, fallback time.Duration) time.Duration {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}
