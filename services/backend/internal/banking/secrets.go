// Package banking owns provider integration, secrets handling and bank connections.
package banking

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// Secret names understood by every SecretSource.
const (
	SecretPlaidClientID = "plaid-client-id"
	SecretPlaidSecret   = "plaid-secret"
	SecretTokenKey      = "token-encryption-key"
)

var ErrSecretUnavailable = errors.New("banking: secret unavailable")

// SecretSource returns raw secret bytes. Errors never contain secret values.
type SecretSource interface {
	Get(ctx context.Context, name string) ([]byte, error)
}

// LocalSource reads files under Dir (default ~/.config/vantage-point/banking); the token key comes from TOKEN_ENCRYPTION_KEY.
type LocalSource struct {
	Dir    string
	Getenv func(string) string
}

func DefaultLocalDir() (string, error) {
	h, err := os.UserHomeDir()
	if err != nil {
		return "", ErrSecretUnavailable
	}
	return filepath.Join(h, ".config", "vantage-point", "banking"), nil
}

func (s LocalSource) Get(_ context.Context, name string) ([]byte, error) {
	if name == SecretTokenKey {
		getenv := s.Getenv
		if getenv == nil {
			getenv = os.Getenv
		}
		v := strings.TrimSpace(getenv("TOKEN_ENCRYPTION_KEY"))
		if v == "" {
			return nil, ErrSecretUnavailable
		}
		return []byte(v), nil
	}
	if strings.ContainsAny(name, `/\`) || name == "" || name == "." || name == ".." {
		return nil, ErrSecretUnavailable
	}
	di, err := os.Stat(s.Dir)
	if err != nil || !di.IsDir() || di.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("%w: secret directory must exist with mode 0700", ErrSecretUnavailable)
	}
	p := filepath.Join(s.Dir, name)
	fi, err := os.Stat(p)
	if err != nil || !fi.Mode().IsRegular() || fi.Mode().Perm()&0o077 != 0 {
		return nil, fmt.Errorf("%w: secret file %q must exist with mode 0600", ErrSecretUnavailable, name)
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, ErrSecretUnavailable
	}
	return b, nil
}

// SecretsManagerAPI is the subset of the AWS client used (faked in tests).
type SecretsManagerAPI interface {
	GetSecretValue(ctx context.Context, in *secretsmanager.GetSecretValueInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

type cached struct {
	v   []byte
	exp time.Time
}

// AWSSource reads `<Prefix><name>` (e.g. vantage-point/prod/plaid-secret) with a short in-memory cache.
type AWSSource struct {
	Client SecretsManagerAPI
	Prefix string
	TTL    time.Duration
	Now    func() time.Time
	mu     sync.Mutex
	cache  map[string]cached
}

// NewAWSSource uses the default AWS credential chain; region comes from the environment/profile.
func NewAWSSource(ctx context.Context, env string) (*AWSSource, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil || cfg.Region == "" {
		return nil, fmt.Errorf("%w: AWS configuration (region) missing", ErrSecretUnavailable)
	}
	return &AWSSource{Client: secretsmanager.NewFromConfig(cfg), Prefix: "vantage-point/" + env + "/", TTL: 5 * time.Minute}, nil
}

func (s *AWSSource) Get(ctx context.Context, name string) ([]byte, error) {
	now := time.Now
	if s.Now != nil {
		now = s.Now
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if c, ok := s.cache[name]; ok && now().Before(c.exp) {
		return append([]byte(nil), c.v...), nil
	}
	id := s.Prefix + name
	out, err := s.Client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{SecretId: &id})
	if err != nil || out == nil {
		return nil, fmt.Errorf("%w: %q", ErrSecretUnavailable, name)
	}
	var v []byte
	switch {
	case out.SecretBinary != nil:
		v = out.SecretBinary
	case out.SecretString != nil:
		v = []byte(*out.SecretString)
	default:
		return nil, fmt.Errorf("%w: %q", ErrSecretUnavailable, name)
	}
	if s.cache == nil {
		s.cache = map[string]cached{}
	}
	s.cache[name] = cached{append([]byte(nil), v...), now().Add(s.TTL)}
	return v, nil
}

// NewSecretSource selects the source from BANKING_SECRET_SOURCE (local|aws); anything else fails closed.
func NewSecretSource(ctx context.Context, getenv func(string) string) (SecretSource, error) {
	switch getenv("BANKING_SECRET_SOURCE") {
	case "local":
		dir := getenv("BANKING_SECRET_DIR")
		if dir == "" {
			var err error
			if dir, err = DefaultLocalDir(); err != nil {
				return nil, err
			}
		}
		return LocalSource{Dir: dir, Getenv: getenv}, nil
	case "aws":
		env := getenv("BANKING_SECRET_ENV")
		if env == "" || strings.ContainsAny(env, "/ ") {
			return nil, fmt.Errorf("%w: BANKING_SECRET_ENV required", ErrSecretUnavailable)
		}
		return NewAWSSource(ctx, env)
	}
	return nil, fmt.Errorf("%w: BANKING_SECRET_SOURCE must be local or aws", ErrSecretUnavailable)
}

// LoadKeyring builds the token keyring (version 1) from the source. The key is base64 of exactly 32 bytes.
func LoadKeyring(ctx context.Context, src SecretSource) (*Keyring, error) {
	raw, err := src.Get(ctx, SecretTokenKey)
	if err != nil {
		return nil, err
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		return nil, ErrBadKey
	}
	return NewKeyring(1, map[byte][]byte{1: key})
}
