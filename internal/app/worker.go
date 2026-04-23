package app

import (
	"context"
	"strings"
	"time"

	"relay/internal/config"
	"relay/internal/lib"
	"relay/internal/services"
)

func RunCuratorWorker(ctx context.Context, cfg config.Config) error {
	runtime, err := NewRuntime(ctx, cfg)
	if err != nil {
		return err
	}

	provider, err := curatorProvider(cfg)
	if err != nil {
		return err
	}
	options := services.CuratorRunOptions{
		Owner:         cfg.CuratorWorkerID,
		BatchSize:     cfg.CuratorBatchSize,
		LeaseDuration: cfg.CuratorLeaseDuration,
		RetryBackoff:  cfg.CuratorRetryBackoff,
		MaxAttempts:   cfg.CuratorMaxAttempts,
	}

	for {
		result, err := runtime.Services.RunCuratorOnce(ctx, provider, options)
		if err != nil {
			return err
		}
		if result.Claimed > 0 {
			continue
		}

		timer := time.NewTimer(cfg.CuratorPollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
}

func curatorProvider(cfg config.Config) (services.CuratorProvider, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.CuratorProvider)) {
	case "", "rule-based":
		return services.RuleBasedCuratorProvider{}, nil
	default:
		return nil, lib.Misconfigured("unsupported RELAY_CURATOR_PROVIDER: " + cfg.CuratorProvider)
	}
}
