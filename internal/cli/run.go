package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"relay/internal/app"
	"relay/internal/config"
	"relay/internal/contracts"
	"relay/internal/services"
	"relay/internal/storage"
)

func Run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	cfg := config.Load()
	if len(args) > 0 && args[0] == "migrate" {
		return runMigrate(stdout, cfg)
	}

	return runWithFactory(args, stdin, stdout, stderr, func() (services.Service, error) {
		runtime, err := app.NewRuntime(context.Background(), cfg)
		if err != nil {
			return services.Service{}, err
		}
		return runtime.Services, nil
	})
}

func runWithFactory(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer, factory func() (services.Service, error)) error {
	if len(args) == 0 {
		_, err := fmt.Fprintln(stdout, usage())
		return err
	}

	switch args[0] {
	case "capture":
		return runCapture(args[1:], stdin, stdout, factory)
	case "migrate":
		_, err := fmt.Fprintln(stdout, usage())
		return err
	case "promote":
		return runPromote(args[1:], stdin, stdout, factory)
	case "packet":
		if len(args) > 1 && args[1] == "build" {
			return runPacketBuild(args[2:], stdin, stdout, factory)
		}
		_, err := fmt.Fprintln(stdout, usage())
		return err
	case "show":
		return runShow(args[1:], stdin, stdout, factory)
	default:
		_, err := fmt.Fprintln(stdout, usage())
		return err
	}
}

func usage() string {
	return `relay commands:
  relay capture
  relay migrate
  relay promote
  relay packet build
  relay show

Common flags:
  --json
  --stdin-json
  --idempotency-key <key>`
}

func runMigrate(stdout io.Writer, cfg config.Config) error {
	db, err := storage.Open(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	if err := storage.ApplyMigrations(context.Background(), db); err != nil {
		return err
	}

	_, err = fmt.Fprintln(stdout, "relay migrations applied")
	return err
}

func runLeaf(command string, args []string, stdin io.Reader, stdout io.Writer) error {
	input, asJSON, err := parseInput(args, stdin)
	if err != nil {
		return err
	}

	if asJSON {
		return json.NewEncoder(stdout).Encode(notImplementedEnvelope(command, input))
	}

	_, err = fmt.Fprintf(stdout, "%s scaffolded; use --json or --stdin-json for agent mode\n", command)
	return err
}

func runCapture(args []string, stdin io.Reader, stdout io.Writer, factory func() (services.Service, error)) error {
	input, asJSON, err := parseInput(args, stdin)
	if err != nil {
		return err
	}
	svc, err := factory()
	if err != nil {
		if asJSON {
			return json.NewEncoder(stdout).Encode(failureEnvelope("relay capture", err))
		}
		return err
	}
	result, err := svc.Capture(context.Background(), decodeCaptureInput(input))
	if err != nil {
		return json.NewEncoder(stdout).Encode(failureEnvelope("relay capture", err))
	}
	if asJSON {
		return json.NewEncoder(stdout).Encode(contracts.Success("relay capture", result))
	}
	_, err = fmt.Fprintf(stdout, "captured project=%s notes=%d artifacts=%d\n", result.ProjectID, len(result.CreatedNoteIDs), len(result.CreatedArtifactIDs))
	return err
}

func runPromote(args []string, stdin io.Reader, stdout io.Writer, factory func() (services.Service, error)) error {
	input, asJSON, err := parseInput(args, stdin)
	if err != nil {
		return err
	}
	svc, err := factory()
	if err != nil {
		if asJSON {
			return json.NewEncoder(stdout).Encode(failureEnvelope("relay promote", err))
		}
		return err
	}
	result, err := svc.Promote(context.Background(), decodePromoteInput(input))
	if err != nil {
		return json.NewEncoder(stdout).Encode(failureEnvelope("relay promote", err))
	}
	if asJSON {
		return json.NewEncoder(stdout).Encode(contracts.Success("relay promote", result))
	}
	_, err = fmt.Fprintf(stdout, "promoted %s=%s project=%s\n", result.Kind, result.ObjectID, result.ProjectID)
	return err
}

func runPacketBuild(args []string, stdin io.Reader, stdout io.Writer, factory func() (services.Service, error)) error {
	input, asJSON, err := parseInput(args, stdin)
	if err != nil {
		return err
	}
	svc, err := factory()
	if err != nil {
		if asJSON {
			return json.NewEncoder(stdout).Encode(failureEnvelope("relay packet build", err))
		}
		return err
	}
	result, err := svc.BuildPacket(context.Background(), decodePacketBuildInput(input))
	if err != nil {
		return json.NewEncoder(stdout).Encode(failureEnvelope("relay packet build", err))
	}
	if asJSON {
		return json.NewEncoder(stdout).Encode(contracts.Success("relay packet build", result))
	}
	_, err = fmt.Fprintf(stdout, "built packet=%s project=%s target=%s\n", result.PacketID, result.ProjectID, result.Target)
	return err
}

func runShow(args []string, stdin io.Reader, stdout io.Writer, factory func() (services.Service, error)) error {
	input, asJSON, err := parseInput(args, stdin)
	if err != nil {
		return err
	}
	svc, err := factory()
	if err != nil {
		if asJSON {
			return json.NewEncoder(stdout).Encode(failureEnvelope("relay show", err))
		}
		return err
	}
	result, err := svc.Show(context.Background(), decodeShowInput(input))
	if err != nil {
		return json.NewEncoder(stdout).Encode(failureEnvelope("relay show", err))
	}
	if asJSON {
		return json.NewEncoder(stdout).Encode(contracts.Success("relay show", result))
	}
	_, err = fmt.Fprintf(stdout, "project=%s notes=%d artifacts=%d decisions=%d open_questions=%d latest_packet=%s\n",
		result.ProjectID, result.NoteCount, result.ArtifactCount, result.DecisionCount, result.OpenQuestionCount, result.LatestPacketID)
	return err
}
