// Package crypto provides envelope encryption (AES-256-GCM) for at-rest
// secrets such as the per-user Anthropic API key. The KEK (key-encryption-key)
// is loaded once at boot from the RELAY_DATA_ENCRYPTION_KEY environment
// variable and held in a version map so V2.5 can rotate without rewriting
// every ciphertext at once. The schema preserves kek_version per row.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"

	"relay/internal/lib"
)

// KEKVersion identifies which key-encryption-key was used to seal an envelope.
// Stored per row as SMALLINT so V2.5 rotation can introduce kek_version=2
// while existing kek_version=1 ciphertexts remain decryptable.
type KEKVersion uint8

// Envelope is a sealed AES-256-GCM ciphertext bound to a specific KEK version.
type Envelope struct {
	Ciphertext []byte
	Nonce      []byte
	KEKVersion KEKVersion
}

// Encrypt seals plaintext under keys[version] using AES-256-GCM. aad is the
// per-row associated-data salt (D6), which authenticates the row binding.
//
// aead.Seal(nil, ...) returns a freshly allocated ciphertext slice — the nil
// dst is canonical Go usage and avoids accidental aliasing with plaintext.
func Encrypt(keys map[KEKVersion][]byte, version KEKVersion, plaintext, aad []byte) (Envelope, error) {
	kek, ok := keys[version]
	if !ok {
		return Envelope{}, fmt.Errorf("kek_version %d not in key map", version)
	}
	if len(kek) != 32 {
		return Envelope{}, fmt.Errorf("kek must be 32 bytes, got %d", len(kek))
	}
	block, err := aes.NewCipher(kek)
	if err != nil {
		return Envelope{}, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return Envelope{}, err
	}
	nonce := make([]byte, aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return Envelope{}, err
	}
	return Envelope{
		Ciphertext: aead.Seal(nil, nonce, plaintext, aad),
		Nonce:      nonce,
		KEKVersion: version,
	}, nil
}

// Decrypt opens an envelope using keys[env.KEKVersion]. An unknown version is a
// MISCONFIGURED error (lib.Misconfigured), not a generic GCM auth error: that
// surfaces "we have a row that this server cannot decrypt" as an operator
// signal instead of looking like a corrupt key (E3, F5).
func Decrypt(keys map[KEKVersion][]byte, env Envelope, aad []byte) ([]byte, error) {
	kek, ok := keys[env.KEKVersion]
	if !ok {
		return nil, lib.Misconfigured(fmt.Sprintf("kek_version %d not in key map; rotation requires offline re-key", env.KEKVersion))
	}
	if len(kek) != 32 {
		return nil, fmt.Errorf("kek must be 32 bytes, got %d", len(kek))
	}
	block, err := aes.NewCipher(kek)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return aead.Open(nil, env.Nonce, env.Ciphertext, aad)
}
