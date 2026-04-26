package crypto

import (
	"encoding/hex"
	"os"
	"strings"

	"relay/internal/lib"
)

// envDataEncryptionKey is the 64-hex-char SealedSecret holding the active KEK.
const envDataEncryptionKey = "RELAY_DATA_ENCRYPTION_KEY"

// LoadKEKsFromEnv reads RELAY_DATA_ENCRYPTION_KEY (64 hex chars = 32 raw bytes)
// and returns a single-entry KEK map keyed by version 1, plus the active
// version. Missing/wrong-length/non-hex input becomes lib.Misconfigured so the
// process fails fast at boot (F1).
func LoadKEKsFromEnv() (map[KEKVersion][]byte, KEKVersion, error) {
	raw := strings.TrimSpace(os.Getenv(envDataEncryptionKey))
	if raw == "" {
		return nil, 0, lib.Misconfigured(envDataEncryptionKey + " is required (64 hex chars)")
	}
	if len(raw) != 64 {
		return nil, 0, lib.Misconfigured(envDataEncryptionKey + ": 64 hex chars required")
	}
	decoded, err := hex.DecodeString(raw)
	if err != nil {
		return nil, 0, lib.Misconfigured(envDataEncryptionKey + ": value must be hex-encoded")
	}
	if len(decoded) != 32 {
		return nil, 0, lib.Misconfigured(envDataEncryptionKey + ": 32 bytes required")
	}
	const active KEKVersion = 1
	return map[KEKVersion][]byte{active: decoded}, active, nil
}
