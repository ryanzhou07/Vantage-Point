package banking

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type ca struct {
	cert *x509.Certificate
	key  *ecdsa.PrivateKey
	pool *x509.CertPool
}

func newCA(t *testing.T) *ca {
	t.Helper()
	k, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "test-ca"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign}
	der, _ := x509.CreateCertificate(rand.Reader, tpl, tpl, &k.PublicKey, k)
	c, _ := x509.ParseCertificate(der)
	p := x509.NewCertPool()
	p.AddCert(c)
	return &ca{c, k, p}
}

// issue returns PEM cert/key for a leaf signed by the CA.
func (a *ca) issue(t *testing.T, notAfter time.Time, server bool) (certPEM, keyPEM []byte) {
	t.Helper()
	k, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tpl := &x509.Certificate{SerialNumber: big.NewInt(time.Now().UnixNano()), Subject: pkix.Name{CommonName: "leaf"}, NotBefore: time.Now().Add(-2 * time.Hour), NotAfter: notAfter, KeyUsage: x509.KeyUsageDigitalSignature}
	if server {
		tpl.IPAddresses = []net.IP{net.ParseIP("127.0.0.1")}
		tpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	} else {
		tpl.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}
	der, _ := x509.CreateCertificate(rand.Reader, tpl, a.cert, &k.PublicKey, a.key)
	kb, _ := x509.MarshalECPrivateKey(k)
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kb})
}

type mapSource map[string]string

func (m mapSource) Get(_ context.Context, n string) ([]byte, error) {
	if v, ok := m[n]; ok {
		return []byte(v), nil
	}
	return nil, ErrSecretUnavailable
}

type fakeTeller struct {
	srv   *httptest.Server
	calls atomic.Int32
	user  atomic.Value
}

func startTeller(t *testing.T, serverCA, clientCA *ca, h func(w http.ResponseWriter, r *http.Request)) *fakeTeller {
	t.Helper()
	f := &fakeTeller{}
	sc, sk := serverCA.issue(t, time.Now().Add(time.Hour), true)
	pair, _ := tls.X509KeyPair(sc, sk)
	f.srv = httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.calls.Add(1)
		u, _, _ := r.BasicAuth()
		f.user.Store(u)
		h(w, r)
	}))
	f.srv.TLS = &tls.Config{Certificates: []tls.Certificate{pair}, ClientAuth: tls.RequireAndVerifyClientCert, ClientCAs: clientCA.pool}
	f.srv.StartTLS()
	t.Cleanup(f.srv.Close)
	return f
}

func ok(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(`[{"id":"acc_1"}]`)) }

func client(t *testing.T, env Env, src SecretSource, f *fakeTeller, roots *x509.CertPool, timeout time.Duration) (*TellerClient, error) {
	c, err := newTellerClient(context.Background(), env, src, f.srv.URL, roots, timeout, time.Millisecond)
	if c != nil {
		c.sleep = func(time.Duration) {}
	}
	return c, err
}

func TestMutualTLS(t *testing.T) {
	a := newCA(t)
	f := startTeller(t, a, a, ok)
	cc, ck := a.issue(t, time.Now().Add(time.Hour), false)
	c, err := client(t, EnvDevelopment, mapSource{SecretTellerCert: string(cc), SecretTellerKey: string(ck)}, f, a.pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Verify(context.Background(), "tok_secret"); err != nil {
		t.Fatal(err)
	}
	if f.user.Load() != "tok_secret" {
		t.Fatal("token not sent as basic username")
	}
	// Sandbox without a certificate must be rejected by a server requiring one.
	nc, err := client(t, EnvSandbox, mapSource{}, f, a.pool, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := nc.Verify(context.Background(), "tok_secret"); !errors.Is(err, ErrProviderUnavailable) {
		t.Fatal("call without client cert succeeded", err)
	}
}

func TestWrongServerCert(t *testing.T) {
	a, other := newCA(t), newCA(t)
	f := startTeller(t, other, a, ok) // server cert from a CA the client does not trust
	cc, ck := a.issue(t, time.Now().Add(time.Hour), false)
	c, _ := client(t, EnvProduction, mapSource{SecretTellerCert: string(cc), SecretTellerKey: string(ck)}, f, a.pool, time.Second)
	if err := c.Verify(context.Background(), "tok"); !errors.Is(err, ErrProviderUnavailable) {
		t.Fatal("untrusted server accepted", err)
	}
	if f.calls.Load() != 0 {
		t.Fatal("request reached untrusted server")
	}
}

func TestStartupFailsClosed(t *testing.T) {
	a := newCA(t)
	f := startTeller(t, a, a, ok)
	good, gk := a.issue(t, time.Now().Add(time.Hour), false)
	_, otherKey := a.issue(t, time.Now().Add(time.Hour), false)
	exp, ek := a.issue(t, time.Now().Add(-time.Minute), false)
	cases := map[string]struct {
		env Env
		src mapSource
	}{
		"dev missing":        {EnvDevelopment, mapSource{}},
		"prod missing":       {EnvProduction, mapSource{}},
		"dev mismatched key": {EnvDevelopment, mapSource{SecretTellerCert: string(good), SecretTellerKey: string(otherKey)}},
		"dev expired":        {EnvDevelopment, mapSource{SecretTellerCert: string(exp), SecretTellerKey: string(ek)}},
		"sandbox expired":    {EnvSandbox, mapSource{SecretTellerCert: string(exp), SecretTellerKey: string(ek)}},
		"sandbox only cert":  {EnvSandbox, mapSource{SecretTellerCert: string(good)}},
		"sandbox garbage":    {EnvSandbox, mapSource{SecretTellerCert: "x", SecretTellerKey: "y"}},
	}
	for name, tc := range cases {
		if c, err := client(t, tc.env, tc.src, f, a.pool, time.Second); err == nil || c != nil {
			t.Errorf("%s: started", name)
		}
	}
	if _, err := client(t, EnvSandbox, mapSource{SecretTellerCert: string(good), SecretTellerKey: string(gk)}, f, a.pool, time.Second); err != nil {
		t.Fatal(err)
	}
}

func TestRetryBoundTimeoutAndErrors(t *testing.T) {
	a := newCA(t)
	cc, ck := a.issue(t, time.Now().Add(time.Hour), false)
	src := mapSource{SecretTellerCert: string(cc), SecretTellerKey: string(ck)}
	run := func(h func(http.ResponseWriter, *http.Request), timeout time.Duration) (*fakeTeller, error) {
		f := startTeller(t, a, a, h)
		c, _ := client(t, EnvDevelopment, src, f, a.pool, timeout)
		return f, c.Verify(context.Background(), "tok_secret")
	}
	if f, err := run(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(500) }, time.Second); !errors.Is(err, ErrProviderUnavailable) || f.calls.Load() != maxAttempts {
		t.Fatal("500 retry bound", err, f.calls.Load())
	}
	if f, err := run(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(429) }, time.Second); !errors.Is(err, ErrProviderRateLimited) || f.calls.Load() != maxAttempts {
		t.Fatal("429", err)
	}
	if f, err := run(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(401) }, time.Second); !errors.Is(err, ErrProviderAuth) || f.calls.Load() != 1 {
		t.Fatal("401 must not retry", err)
	}
	if _, err := run(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(404) }, time.Second); !errors.Is(err, ErrProviderRequest) {
		t.Fatal("404", err)
	}
	if f, err := run(func(w http.ResponseWriter, _ *http.Request) { time.Sleep(300 * time.Millisecond) }, 50*time.Millisecond); !errors.Is(err, ErrProviderUnavailable) || f.calls.Load() > maxAttempts {
		t.Fatal("timeout", err)
	}
	if _, err := run(func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(`{"x":1}`)) }, time.Second); !errors.Is(err, ErrProviderResponse) {
		t.Fatal("shape", err)
	}
	if _, err := run(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("[\"" + strings.Repeat("a", maxResponseSize) + "\"]"))
	}, time.Second); !errors.Is(err, ErrProviderResponse) {
		t.Fatal("oversize", err)
	}
	_, err := run(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(500) }, time.Second)
	if strings.Contains(err.Error(), "tok_secret") || strings.Contains(err.Error(), "127.0.0.1") {
		t.Fatal("error leaks")
	}
}

func TestParseEnv(t *testing.T) {
	if e, err := ParseEnv(""); err != nil || e != EnvSandbox {
		t.Fatal("default")
	}
	if _, err := ParseEnv("live"); err == nil {
		t.Fatal("invalid accepted")
	}
}
