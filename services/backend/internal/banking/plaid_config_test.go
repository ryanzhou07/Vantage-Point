package banking

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
)

type mapSource map[string]string

func (m mapSource) Get(_ context.Context, n string) ([]byte, error) {
	if v, ok := m[n]; ok {
		return []byte(v), nil
	}
	return nil, ErrSecretUnavailable
}

func TestLoadPlaidConfig(t *testing.T) {
	full := mapSource{SecretPlaidClientID: "cid-123", SecretPlaidSecret: "sek-456\n"}
	env := func(v string) func(string) string { return func(string) string { return v } }
	for _, tc := range []struct {
		name, env string
		src       mapSource
		wantEnv   string
		wantErr   bool
	}{
		{"default sandbox", "", full, "sandbox", false},
		{"production", "production", full, "production", false},
		{"unknown env", "prod", full, "", true},
		{"missing secret", "production", mapSource{SecretPlaidClientID: "x"}, "", true},
		{"missing client id", "sandbox", mapSource{SecretPlaidSecret: "x"}, "", true},
		{"empty creds", "sandbox", mapSource{SecretPlaidClientID: " ", SecretPlaidSecret: "x"}, "", true},
	} {
		c, err := LoadPlaidConfig(context.Background(), env(tc.env), tc.src)
		if (err != nil) != tc.wantErr || c.Env != tc.wantEnv {
			t.Fatalf("%s: got %v env=%q", tc.name, err, c.Env)
		}
		if err != nil && !errors.Is(err, ErrBadPlaidConfig) {
			t.Fatalf("%s: wrong error %v", tc.name, err)
		}
	}
	c, _ := LoadPlaidConfig(context.Background(), env(""), full)
	if c.Secret != "sek-456" {
		t.Fatal("secret not trimmed")
	}
	if s := fmt.Sprintf("%v %+v %#v", c, c, c); strings.Contains(s, "cid-123") || strings.Contains(s, "sek-456") {
		t.Fatalf("credentials leaked: %s", s)
	}
}
