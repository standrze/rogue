package proxy

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/standrze/rogue/internal/cert"
	"github.com/standrze/rogue/internal/logger"
)

func TestNewProxyServer(t *testing.T) {
	// Setup temporary directory for certs and logs
	tmpDir, err := os.MkdirTemp("", "rogue-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	certPath := filepath.Join(tmpDir, "ca.crt")
	keyPath := filepath.Join(tmpDir, "ca.key")
	sessionDir := filepath.Join(tmpDir, "logs")

	// Create proxy
	proxy := NewProxyServer(
		WithCert(certPath, keyPath),
		WithSessionDir(sessionDir),
	)

	// Start proxy listener
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	proxyPort := l.Addr().(*net.TCPAddr).Port
	go proxy.Serve(l)

	// Wait for proxy to start
	time.Sleep(100 * time.Millisecond)

	// Check if certs were generated
	if !cert.Exists(certPath, keyPath) {
		t.Error("Certs were not generated")
	}

	// Make a request through the proxy
	proxyURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", proxyPort))
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, // We are using self-signed certs
			},
		},
	}

	resp, err := client.Get("http://example.com")
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Check if logs were created
	sessions, err := logger.ListSessions(sessionDir)
	if err != nil {
		t.Fatalf("Failed to list sessions: %v", err)
	}
	if len(sessions) == 0 {
		t.Error("No session logs found")
	}

	// Test Export to Markdown
	if len(sessions) > 0 {
		sessionName := sessions[0]
		mdPath := filepath.Join(tmpDir, "session.md")
		err := logger.ExportSessionToMarkdown(sessionDir, sessionName, mdPath)
		if err != nil {
			t.Errorf("Failed to export markdown: %v", err)
		}
		if _, err := os.Stat(mdPath); os.IsNotExist(err) {
			t.Error("Markdown file not created")
		}
	}
}
