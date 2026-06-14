package certstore

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type FetchOptions struct {
	Port    string
	Timeout time.Duration
	Chain   bool
	Dir     string
}

type FetchResult struct {
	Host  string
	Port  string
	Files []Certificate
}

func FetchFromTLS(rawURL string, opts FetchOptions) (FetchResult, error) {
	if strings.TrimSpace(rawURL) == "" {
		return FetchResult{}, fmt.Errorf("informe a URL ou host do servidor")
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 15 * time.Second
	}

	host, port, err := parseTLSHost(rawURL, opts.Port)
	if err != nil {
		return FetchResult{}, err
	}

	dir := strings.TrimSpace(opts.Dir)
	if dir == "" {
		dir, err = Directory()
		if err != nil {
			return FetchResult{}, err
		}
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return FetchResult{}, err
	}

	address := net.JoinHostPort(host, port)
	dialer := &net.Dialer{Timeout: opts.Timeout}
	conn, err := tls.DialWithDialer(dialer, "tcp", address, &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: true,
	})
	if err != nil {
		return FetchResult{}, fmt.Errorf("falha ao conectar em %s: %w", address, err)
	}
	defer conn.Close()

	peerCerts := conn.ConnectionState().PeerCertificates
	if len(peerCerts) == 0 {
		return FetchResult{}, fmt.Errorf("nenhum certificado retornado por %s", address)
	}

	result := FetchResult{Host: host, Port: port}
	baseName := safeName(host)

	leafPath := filepath.Join(dir, baseName+".pem")
	if err := writeCertificatePEM(leafPath, peerCerts[0]); err != nil {
		return FetchResult{}, err
	}
	result.Files = append(result.Files, Certificate{Path: leafPath, Name: baseName + ".pem"})

	if opts.Chain && len(peerCerts) > 1 {
		chainPath := filepath.Join(dir, baseName+"-chain.pem")
		if err := writeCertificateChainPEM(chainPath, peerCerts); err != nil {
			return FetchResult{}, err
		}
		result.Files = append(result.Files, Certificate{Path: chainPath, Name: baseName + "-chain.pem"})
	}

	return result, nil
}

func parseTLSHost(rawURL string, portOverride string) (string, string, error) {
	value := strings.TrimSpace(rawURL)
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}

	parsed, err := url.Parse(value)
	if err != nil {
		return "", "", err
	}
	host := parsed.Hostname()
	if host == "" {
		return "", "", fmt.Errorf("host invalido na URL: %s", rawURL)
	}

	port := parsed.Port()
	if portOverride != "" {
		port = portOverride
	}
	if port == "" {
		switch strings.ToLower(parsed.Scheme) {
		case "http":
			port = "80"
		default:
			port = "443"
		}
	}

	return host, port, nil
}

func writeCertificatePEM(path string, cert *x509.Certificate) error {
	return writePEMBlock(path, "CERTIFICATE", cert.Raw)
}

func writeCertificateChainPEM(path string, certs []*x509.Certificate) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	for _, cert := range certs {
		if err := pem.Encode(file, &pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw}); err != nil {
			return err
		}
	}
	return nil
}

func writePEMBlock(path string, blockType string, data []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	return pem.Encode(file, &pem.Block{Type: blockType, Bytes: data})
}
