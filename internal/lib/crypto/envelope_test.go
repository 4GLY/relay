package crypto

import (
	"bytes"
	"strings"
	"testing"

	"relay/internal/lib"
)

func newKey32(b byte) []byte {
	out := make([]byte, 32)
	for i := range out {
		out[i] = b
	}
	return out
}

// T1: roundtrip with the correct AAD succeeds; decrypting with a different
// AAD fails GCM auth — the row binding is enforced cryptographically.
func TestEnvelopeAADBindingRoundtrip(t *testing.T) {
	keys := map[KEKVersion][]byte{1: newKey32(0xab)}
	plaintext := []byte("sk-ant-fake-key-value")
	saltA := bytes.Repeat([]byte{0x01}, 16)
	saltB := bytes.Repeat([]byte{0x02}, 16)

	env, err := Encrypt(keys, 1, plaintext, saltA)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	got, err := Decrypt(keys, env, saltA)
	if err != nil {
		t.Fatalf("Decrypt with correct AAD failed: %v", err)
	}
	if !bytes.Equal(got, plaintext) {
		t.Fatalf("decrypted plaintext mismatch: got %q want %q", got, plaintext)
	}

	if _, err := Decrypt(keys, env, saltB); err == nil {
		t.Fatal("expected Decrypt with mismatched AAD to fail GCM auth, got nil error")
	}
}

// T11: a row encrypted under kek_version=1 cannot be decrypted by a service
// that only has kek_version=2 in its key map. The error is lib.Misconfigured
// (E3, F5) — distinct from a generic GCM auth failure so operators can spot
// the rotation cliff.
func TestDecryptUnknownKEKVersionIsMisconfigured(t *testing.T) {
	keysV1 := map[KEKVersion][]byte{1: newKey32(0xcc)}
	keysV2 := map[KEKVersion][]byte{2: newKey32(0xdd)}
	salt := bytes.Repeat([]byte{0x05}, 16)

	env, err := Encrypt(keysV1, 1, []byte("plain"), salt)
	if err != nil {
		t.Fatalf("Encrypt v1 failed: %v", err)
	}

	_, err = Decrypt(keysV2, env, salt)
	if err == nil {
		t.Fatal("expected Decrypt to fail with unknown kek_version")
	}
	appErr, ok := err.(lib.AppError)
	if !ok {
		t.Fatalf("expected lib.AppError, got %T: %v", err, err)
	}
	if appErr.Code != "MISCONFIGURED" {
		t.Fatalf("expected MISCONFIGURED, got %q", appErr.Code)
	}
	if !strings.Contains(appErr.Message, "kek_version 1") {
		t.Fatalf("expected message to name the missing version, got %q", appErr.Message)
	}
}

func TestEncryptRejectsUnknownVersion(t *testing.T) {
	keys := map[KEKVersion][]byte{1: newKey32(0xee)}
	if _, err := Encrypt(keys, 7, []byte("p"), nil); err == nil {
		t.Fatal("expected Encrypt to fail for unknown version 7")
	}
}

func TestEncryptRejectsWrongLengthKey(t *testing.T) {
	keys := map[KEKVersion][]byte{1: bytes.Repeat([]byte{0x11}, 16)}
	if _, err := Encrypt(keys, 1, []byte("p"), nil); err == nil {
		t.Fatal("expected Encrypt to fail for 16-byte key")
	}
}
