package banking

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

func key(b byte) []byte { return bytes.Repeat([]byte{b}, 32) }

func ring(t *testing.T) *Keyring {
	t.Helper()
	k, err := NewKeyring(1, map[byte][]byte{1: key(1), 2: key(2)})
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func TestSealOpen(t *testing.T) {
	k := ring(t)
	blob, _ := k.Seal("u1", "c1", []byte("tok"))
	if got, err := k.Open("u1", "c1", blob); err != nil || string(got) != "tok" {
		t.Fatal(got, err)
	}
	if _, err := k.Open("u2", "c1", blob); !errors.Is(err, ErrDecrypt) {
		t.Fatal("wrong user AAD opened")
	}
	if _, err := k.Open("u1", "c2", blob); !errors.Is(err, ErrDecrypt) {
		t.Fatal("wrong conn AAD opened")
	}
	bad := append([]byte(nil), blob...)
	bad[len(bad)-1] ^= 1
	if _, err := k.Open("u1", "c1", bad); !errors.Is(err, ErrDecrypt) {
		t.Fatal("tampered opened")
	}
	other, _ := NewKeyring(1, map[byte][]byte{1: key(9)})
	if _, err := other.Open("u1", "c1", blob); !errors.Is(err, ErrDecrypt) {
		t.Fatal("wrong key opened")
	}
	ver := append([]byte(nil), blob...)
	ver[0] = 2
	if _, err := k.Open("u1", "c1", ver); !errors.Is(err, ErrDecrypt) {
		t.Fatal("wrong version opened")
	}
	ver[0] = 7
	if _, err := k.Open("u1", "c1", ver); !errors.Is(err, ErrUnknownKey) {
		t.Fatal("unknown version")
	}
	if _, err := k.Open("u1", "c1", []byte{1, 2}); err == nil {
		t.Fatal("short blob")
	}
}

func TestNonceUnique(t *testing.T) {
	k := ring(t)
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		b, _ := k.Seal("u", "c", []byte("x"))
		n := string(b[1:13])
		if seen[n] {
			t.Fatal("nonce reuse")
		}
		seen[n] = true
	}
}

func TestKeyringFailsClosed(t *testing.T) {
	if _, err := NewKeyring(1, map[byte][]byte{1: make([]byte, 16)}); !errors.Is(err, ErrBadKey) {
		t.Fatal("short key accepted")
	}
	if _, err := NewKeyring(3, map[byte][]byte{1: key(1)}); err == nil {
		t.Fatal("missing current accepted")
	}
	src := LocalSource{Getenv: func(string) string { return "" }}
	if _, err := LoadKeyring(context.Background(), src); err == nil {
		t.Fatal("missing key accepted")
	}
	short := LocalSource{Getenv: func(string) string { return base64.StdEncoding.EncodeToString(make([]byte, 16)) }}
	if _, err := LoadKeyring(context.Background(), short); !errors.Is(err, ErrBadKey) {
		t.Fatal("short env key accepted")
	}
	ok := LocalSource{Getenv: func(string) string { return base64.StdEncoding.EncodeToString(key(5)) }}
	if _, err := LoadKeyring(context.Background(), ok); err != nil {
		t.Fatal(err)
	}
}

func TestLocalPermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "teller")
	os.Mkdir(dir, 0o700)
	f := filepath.Join(dir, "teller-cert")
	os.WriteFile(f, []byte("PEM"), 0o600)
	s := LocalSource{Dir: dir}
	if b, err := s.Get(context.Background(), SecretTellerCert); err != nil || string(b) != "PEM" {
		t.Fatal(err)
	}
	os.Chmod(f, 0o644)
	if _, err := s.Get(context.Background(), SecretTellerCert); err == nil {
		t.Fatal("loose file accepted")
	}
	os.Chmod(f, 0o600)
	os.Chmod(dir, 0o755)
	if _, err := s.Get(context.Background(), SecretTellerCert); err == nil {
		t.Fatal("loose dir accepted")
	}
	os.Chmod(dir, 0o700)
	if _, err := s.Get(context.Background(), "../x"); err == nil {
		t.Fatal("traversal accepted")
	}
}

type fakeSM struct {
	calls int
	err   error
}

func (f *fakeSM) GetSecretValue(_ context.Context, in *secretsmanager.GetSecretValueInput, _ ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	v := "val:" + *in.SecretId
	return &secretsmanager.GetSecretValueOutput{SecretString: &v}, nil
}

func TestAWSSourceCacheAndRedaction(t *testing.T) {
	now := time.Unix(0, 0)
	f := &fakeSM{}
	s := &AWSSource{Client: f, Prefix: "vantage-point/dev/", TTL: time.Minute, Now: func() time.Time { return now }}
	b, err := s.Get(context.Background(), SecretTellerCert)
	if err != nil || string(b) != "val:vantage-point/dev/teller-cert" {
		t.Fatal(string(b), err)
	}
	s.Get(context.Background(), SecretTellerCert)
	if f.calls != 1 {
		t.Fatal("not cached")
	}
	now = now.Add(2 * time.Minute)
	s.Get(context.Background(), SecretTellerCert)
	if f.calls != 2 {
		t.Fatal("not expired")
	}
	f.err = errors.New("AKIASECRETLEAK")
	_, err = (&AWSSource{Client: f}).Get(context.Background(), SecretTellerKey)
	if err == nil || strings.Contains(err.Error(), "AKIA") {
		t.Fatal("error leaks or missing", err)
	}
}

func TestNewSecretSourceFailsClosed(t *testing.T) {
	if _, err := NewSecretSource(context.Background(), func(string) string { return "" }); err == nil {
		t.Fatal("unset source accepted")
	}
	if _, err := NewSecretSource(context.Background(), func(k string) string {
		return map[string]string{"TELLER_SECRET_SOURCE": "aws"}[k]
	}); err == nil {
		t.Fatal("aws without env accepted")
	}
}
