package lib

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

func StableID(prefix string, seed string) string {
	hash := sha256.Sum256([]byte(seed))
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(hash[:])[:12])
}

func NewID(prefix string) string {
	return StableID(prefix, fmt.Sprintf("%s:%d", prefix, time.Now().UnixNano()))
}

func ProjectID(name string) string {
	return StableID("proj", strings.ToLower(strings.TrimSpace(name)))
}
