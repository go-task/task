package taskfile

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-task/task/v3/errors"
)

func TestHTTPNode_CacheKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		entrypoint  string
		expectedKey string
	}{
		{
			entrypoint:  "https://github.com",
			expectedKey: "http.github.com..996e1f714b08e971ec79e3bea686287e66441f043177999a13dbc546d8fe402a",
		},
		{
			entrypoint:  "https://github.com/Taskfile.yml",
			expectedKey: "http.github.com.Taskfile.yml.85b3c3ad71b78dc74e404c7b4390fc13672925cb644a4d26c21b9f97c17b5fc0",
		},
		{
			entrypoint:  "https://github.com/foo",
			expectedKey: "http.github.com.foo.df3158dafc823e6847d9bcaf79328446c4877405e79b100723fa6fd545ed3e2b",
		},
		{
			entrypoint:  "https://github.com/foo/Taskfile.yml",
			expectedKey: "http.github.com.foo.Taskfile.yml.aea946ea7eb6f6bb4e159e8b840b6b50975927778b2e666df988c03bbf10c4c4",
		},
		{
			entrypoint:  "https://github.com/foo/bar",
			expectedKey: "http.github.com.foo.bar.d3514ad1d4daedf9cc2825225070b49ebc8db47fa5177951b2a5b9994597570c",
		},
		{
			entrypoint:  "https://github.com/foo/bar/Taskfile.yml",
			expectedKey: "http.github.com.bar.Taskfile.yml.b9cf01e01e47c0e96ea536e1a8bd7b3a6f6c1f1881bad438990d2bfd4ccd0ac0",
		},
	}

	for _, tt := range tests {
		node, err := NewHTTPNode(tt.entrypoint, "", false)
		require.NoError(t, err)
		key := node.CacheKey()
		assert.Equal(t, tt.expectedKey, key)
	}
}

func TestBuildHTTPClient_Default(t *testing.T) {
	t.Parallel()

	// When no TLS customization is needed, should copy http.DefaultClient
	client, err := buildHTTPClient(false, "", "", "")
	require.NoError(t, err)
	assert.NotSame(t, http.DefaultClient, client)
	assert.Equal(t, http.DefaultClient.Transport, client.Transport)
	assert.NotNil(t, client.CheckRedirect)
	// The shared client must keep following redirects as before.
	assert.Nil(t, http.DefaultClient.CheckRedirect)
}

func TestBuildHTTPClient_Insecure(t *testing.T) {
	t.Parallel()

	client, err := buildHTTPClient(true, "", "", "")
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.NotEqual(t, http.DefaultClient, client)

	// Check that InsecureSkipVerify is set
	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	require.NotNil(t, transport.TLSClientConfig)
	assert.True(t, transport.TLSClientConfig.InsecureSkipVerify)
}

func TestBuildHTTPClient_CACert(t *testing.T) {
	t.Parallel()

	// Create a temporary CA cert file
	tempDir := t.TempDir()
	caCertPath := filepath.Join(tempDir, "ca.crt")

	// Generate a valid CA certificate
	caCertPEM := generateTestCACert(t)
	err := os.WriteFile(caCertPath, caCertPEM, 0o600)
	require.NoError(t, err)

	client, err := buildHTTPClient(false, caCertPath, "", "")
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.NotEqual(t, http.DefaultClient, client)

	// Check that custom RootCAs is set
	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	require.NotNil(t, transport.TLSClientConfig)
	assert.NotNil(t, transport.TLSClientConfig.RootCAs)
}

func TestBuildHTTPClient_CACertNotFound(t *testing.T) {
	t.Parallel()

	client, err := buildHTTPClient(false, "/nonexistent/ca.crt", "", "")
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to read CA certificate")
}

func TestBuildHTTPClient_CACertInvalid(t *testing.T) {
	t.Parallel()

	// Create a temporary file with invalid content
	tempDir := t.TempDir()
	caCertPath := filepath.Join(tempDir, "invalid.crt")
	err := os.WriteFile(caCertPath, []byte("not a valid certificate"), 0o600)
	require.NoError(t, err)

	client, err := buildHTTPClient(false, caCertPath, "", "")
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to parse CA certificate")
}

func TestBuildHTTPClient_CertWithoutKey(t *testing.T) {
	t.Parallel()

	client, err := buildHTTPClient(false, "", "/path/to/cert.crt", "")
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "both --cert and --cert-key must be provided together")
}

func TestBuildHTTPClient_KeyWithoutCert(t *testing.T) {
	t.Parallel()

	client, err := buildHTTPClient(false, "", "", "/path/to/key.pem")
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "both --cert and --cert-key must be provided together")
}

func TestBuildHTTPClient_CertAndKey(t *testing.T) {
	t.Parallel()

	// Create temporary cert and key files
	tempDir := t.TempDir()
	certPath := filepath.Join(tempDir, "client.crt")
	keyPath := filepath.Join(tempDir, "client.key")

	// Generate a self-signed certificate and key for testing
	cert, key := generateTestCertAndKey(t)
	err := os.WriteFile(certPath, cert, 0o600)
	require.NoError(t, err)
	err = os.WriteFile(keyPath, key, 0o600)
	require.NoError(t, err)

	client, err := buildHTTPClient(false, "", certPath, keyPath)
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.NotEqual(t, http.DefaultClient, client)

	// Check that client certificate is set
	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	require.NotNil(t, transport.TLSClientConfig)
	assert.Len(t, transport.TLSClientConfig.Certificates, 1)
}

func TestBuildHTTPClient_CertNotFound(t *testing.T) {
	t.Parallel()

	client, err := buildHTTPClient(false, "", "/nonexistent/cert.crt", "/nonexistent/key.pem")
	assert.Error(t, err)
	assert.Nil(t, client)
	assert.Contains(t, err.Error(), "failed to load client certificate")
}

func TestBuildHTTPClient_InsecureWithCACert(t *testing.T) {
	t.Parallel()

	// Create a temporary CA cert file
	tempDir := t.TempDir()
	caCertPath := filepath.Join(tempDir, "ca.crt")

	// Generate a valid CA certificate
	caCertPEM := generateTestCACert(t)
	err := os.WriteFile(caCertPath, caCertPEM, 0o600)
	require.NoError(t, err)

	// Both insecure and CA cert can be set together
	client, err := buildHTTPClient(true, caCertPath, "", "")
	require.NoError(t, err)
	require.NotNil(t, client)

	transport, ok := client.Transport.(*http.Transport)
	require.True(t, ok)
	require.NotNil(t, transport.TLSClientConfig)
	assert.True(t, transport.TLSClientConfig.InsecureSkipVerify)
	assert.NotNil(t, transport.TLSClientConfig.RootCAs)
}

// generateTestCertAndKey generates a self-signed certificate and key for testing
func generateTestCertAndKey(t *testing.T) (certPEM, keyPEM []byte) {
	t.Helper()

	// Generate a new ECDSA private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// Create a certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Task Org"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	// Create the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	// Encode certificate to PEM
	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key to PEM
	keyDER, err := x509.MarshalECPrivateKey(privateKey)
	require.NoError(t, err)
	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: keyDER,
	})

	return certPEM, keyPEM
}

// generateTestCACert generates a self-signed CA certificate for testing
func generateTestCACert(t *testing.T) []byte {
	t.Helper()

	// Generate a new ECDSA private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// Create a CA certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test CA"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	// Create the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	require.NoError(t, err)

	// Encode certificate to PEM
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})
}

func TestCheckRedirect(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		from    string
		to      string
		wantErr bool
	}{
		{name: "https to http is refused", from: "https://example.com", to: "http://example.com", wantErr: true},
		{name: "https to http on another host is refused", from: "https://example.com", to: "http://evil.test", wantErr: true},
		{name: "https to https is allowed", from: "https://example.com", to: "https://other.example.com"},
		{name: "http to https is allowed", from: "http://example.com", to: "https://example.com"},
		{name: "http to http is allowed, the entrypoint already opted in", from: "http://example.com", to: "http://other.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			via := []*http.Request{mustGet(t, tt.from)}
			err := checkRedirect(mustGet(t, tt.to), via)
			if !tt.wantErr {
				require.NoError(t, err)
				return
			}
			var notSecure *errors.TaskfileNotSecureError
			require.ErrorAs(t, err, &notSecure)
		})
	}
}

func TestCheckRedirectFirstRequest(t *testing.T) {
	t.Parallel()

	require.NoError(t, checkRedirect(mustGet(t, "http://example.com"), nil))
}

func TestCheckRedirectStopsAfterTenHops(t *testing.T) {
	t.Parallel()

	via := make([]*http.Request, 10)
	for i := range via {
		via[i] = mustGet(t, "https://example.com")
	}
	require.Error(t, checkRedirect(mustGet(t, "https://example.com"), via))
}

// The downgrade is refused even with --insecure, which here also makes the
// client accept the test server's self-signed certificate.
func TestBuildHTTPClientRefusesDowngradeWithInsecure(t *testing.T) {
	t.Parallel()

	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer plain.Close()

	secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, plain.URL+"/Taskfile.yml", http.StatusFound)
	}))
	defer secure.Close()

	client, err := buildHTTPClient(true, "", "", "")
	require.NoError(t, err)

	_, err = client.Do(mustGet(t, secure.URL+"/Taskfile.yml")) //nolint:bodyclose // the request never completes
	var notSecure *errors.TaskfileNotSecureError
	require.ErrorAs(t, err, &notSecure)
}

func mustGet(t *testing.T, rawURL string) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, rawURL, nil)
	require.NoError(t, err)
	return req
}
