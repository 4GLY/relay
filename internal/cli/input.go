package cli

import (
	"encoding/json"
	"errors"
	"io"
)

func parseInput(args []string, stdin io.Reader) (map[string]any, bool, error) {
	var stdinJSON bool
	var asJSON bool

	for _, arg := range args {
		switch arg {
		case "--stdin-json":
			stdinJSON = true
		case "--json":
			asJSON = true
		}
	}

	if !stdinJSON {
		return map[string]any{}, asJSON, nil
	}

	raw, err := io.ReadAll(stdin)
	if err != nil {
		return nil, false, err
	}
	if len(raw) == 0 {
		return nil, false, errors.New("stdin-json requested but stdin was empty")
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, false, err
	}

	return payload, true, nil
}
