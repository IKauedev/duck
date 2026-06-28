package certstore

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseTLSHost(t *testing.T) {
	tests := []struct {
		url      string
		port     string
		wantHost string
		wantPort string
	}{
		{"https://example.com", "", "example.com", "443"},
		{"example.com", "", "example.com", "443"},
		{"https://api.local:8443/path", "", "api.local", "8443"},
		{"api.local", "9443", "api.local", "9443"},
		{"http://legacy.local", "", "legacy.local", "80"},
	}

	for _, tc := range tests {
		host, port, err := parseTLSHost(tc.url, tc.port)
		if err != nil {
			t.Fatalf("parseTLSHost(%q) error: %v", tc.url, err)
		}
		if host != tc.wantHost || port != tc.wantPort {
			t.Fatalf("parseTLSHost(%q) = (%q, %q), want (%q, %q)", tc.url, host, port, tc.wantHost, tc.wantPort)
		}
	}
}

func TestWriteCertificatePEM(t *testing.T) {
	dir := t.TempDir()
	cert := testCertificate(t)
	path := filepath.Join(dir, "test.pem")

	if err := writeCertificatePEM(path, cert); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(data)
	if block == nil || block.Type != "CERTIFICATE" {
		t.Fatalf("arquivo PEM invalido: %s", string(data))
	}
}

func testCertificate(t *testing.T) *x509.Certificate {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test.local",
		},
		NotBefore: time.Now().Add(-time.Hour),
		NotAfter:  time.Now().Add(time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}

	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return cert
}
