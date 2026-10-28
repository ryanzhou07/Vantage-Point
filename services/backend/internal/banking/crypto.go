package banking

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
)

var (
	ErrBadKey     = errors.New("banking: token key must be exactly 32 bytes")
	ErrDecrypt    = errors.New("banking: cannot decrypt credential")
	ErrUnknownKey = errors.New("banking: unknown key version")
)

// Keyring seals with the current key version and opens any known version.
type Keyring struct {
	current byte
	aeads   map[byte]cipher.AEAD
}

func NewKeyring(current byte, keys map[byte][]byte) (*Keyring, error) {
	k := &Keyring{current: current, aeads: map[byte]cipher.AEAD{}}
	for v, key := range keys {
		if len(key) != 32 {
			return nil, ErrBadKey
		}
		b, err := aes.NewCipher(key)
		if err != nil {
			return nil, ErrBadKey
		}
		if k.aeads[v], err = cipher.NewGCM(b); err != nil {
			return nil, ErrBadKey
		}
	}
	if k.aeads[current] == nil {
		return nil, ErrUnknownKey
	}
	return k, nil
}

// aad binds ciphertext to its owner and connection; the length prefix prevents ambiguity.
func aad(userID, connID string) []byte {
	return append(append([]byte{byte(len(userID))}, userID...), connID...)
}

// Version returns the key version used by Seal; store it beside the blob.
func (k *Keyring) Version() byte { return k.current }

// Seal returns version || nonce(12) || ciphertext+tag.
func (k *Keyring) Seal(userID, connID string, plaintext []byte) ([]byte, error) {
	if len(userID) > 255 {
		return nil, ErrDecrypt
	}
	g := k.aeads[k.current]
	out := make([]byte, 1+g.NonceSize(), 1+g.NonceSize()+len(plaintext)+g.Overhead())
	out[0] = k.current
	if _, err := rand.Read(out[1:]); err != nil {
		return nil, err
	}
	return g.Seal(out, out[1:1+g.NonceSize()], plaintext, aad(userID, connID)), nil
}

func (k *Keyring) Open(userID, connID string, blob []byte) ([]byte, error) {
	if len(blob) < 1 || len(userID) > 255 {
		return nil, ErrDecrypt
	}
	g := k.aeads[blob[0]]
	if g == nil {
		return nil, ErrUnknownKey
	}
	n := g.NonceSize()
	if len(blob) < 1+n+g.Overhead() {
		return nil, ErrDecrypt
	}
	pt, err := g.Open(nil, blob[1:1+n], blob[1+n:], aad(userID, connID))
	if err != nil {
		return nil, ErrDecrypt
	}
	return pt, nil
}
