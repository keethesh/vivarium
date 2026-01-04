// Package smoke provides anonymity layer with proxy/Tor support.
// "Beekeepers use smoke to calm and obscure" - your anonymity layer.
package smoke

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

// Config holds proxy configuration.
type Config struct {
	Enabled     bool     // Whether proxy is enabled
	ProxyURL    string   // SOCKS5 proxy URL (e.g., socks5://127.0.0.1:9050)
	ProxyChain  []string // Multiple proxies for chaining
	RotateEvery int      // Rotate circuit every N requests (Tor)
	Timeout     time.Duration
}

// DefaultTorConfig returns config for local Tor proxy.
func DefaultTorConfig() *Config {
	return &Config{
		Enabled:     true,
		ProxyURL:    "socks5://127.0.0.1:9050",
		RotateEvery: 10,
		Timeout:     30 * time.Second,
	}
}

// Smoker manages proxied connections.
type Smoker struct {
	config       *Config
	dialer       proxy.Dialer
	requestCount int
	mu           sync.Mutex
}

// NewSmoker creates a new Smoker with the given config.
func NewSmoker(config *Config) (*Smoker, error) {
	if config == nil || !config.Enabled {
		return &Smoker{config: &Config{Enabled: false}}, nil
	}

	dialer, err := createDialer(config.ProxyURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy dialer: %w", err)
	}

	return &Smoker{
		config: config,
		dialer: dialer,
	}, nil
}

// createDialer creates a SOCKS5 dialer from URL.
func createDialer(proxyURL string) (proxy.Dialer, error) {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, err
	}

	var auth *proxy.Auth
	if u.User != nil {
		password, _ := u.User.Password()
		auth = &proxy.Auth{
			User:     u.User.Username(),
			Password: password,
		}
	}

	dialer, err := proxy.SOCKS5("tcp", u.Host, auth, proxy.Direct)
	if err != nil {
		return nil, err
	}

	return dialer, nil
}

// IsEnabled returns whether proxy is enabled.
func (s *Smoker) IsEnabled() bool {
	return s.config != nil && s.config.Enabled
}

// Dial creates a proxied connection.
func (s *Smoker) Dial(network, addr string) (net.Conn, error) {
	if !s.IsEnabled() {
		return net.Dial(network, addr)
	}
	return s.dialer.Dial(network, addr)
}

// DialContext creates a proxied connection with context.
func (s *Smoker) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	if !s.IsEnabled() {
		var d net.Dialer
		return d.DialContext(ctx, network, addr)
	}

	// Check context before dialing
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// proxy.Dialer doesn't support context, so we dial and check
	conn, err := s.dialer.Dial(network, addr)
	if err != nil {
		return nil, err
	}

	// Check if context was cancelled during dial
	select {
	case <-ctx.Done():
		conn.Close()
		return nil, ctx.Err()
	default:
		return conn, nil
	}
}

// HTTPClient returns an HTTP client configured to use the proxy.
func (s *Smoker) HTTPClient() *http.Client {
	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 100,
		IdleConnTimeout:     90 * time.Second,
	}

	if s.IsEnabled() {
		transport.DialContext = s.DialContext
	}

	timeout := 30 * time.Second
	if s.config != nil && s.config.Timeout > 0 {
		timeout = s.config.Timeout
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// FastHTTPClient returns a high-performance HTTP client for attacks.
func (s *Smoker) FastHTTPClient() *http.Client {
	transport := &http.Transport{
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 1000,
		MaxConnsPerHost:     1000,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false,
		DisableCompression:  true,
	}

	if s.IsEnabled() {
		transport.DialContext = s.DialContext
	}

	timeout := 10 * time.Second
	if s.config != nil && s.config.Timeout > 0 {
		timeout = s.config.Timeout
	}

	return &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// TestConnection tests the proxy connection.
func (s *Smoker) TestConnection(ctx context.Context) error {
	if !s.IsEnabled() {
		return nil
	}

	// Try to connect to a known host
	conn, err := s.DialContext(ctx, "tcp", "1.1.1.1:80")
	if err != nil {
		return fmt.Errorf("proxy connection failed: %w", err)
	}
	conn.Close()

	return nil
}

// GetExternalIP gets your external IP through the proxy.
func (s *Smoker) GetExternalIP(ctx context.Context) (string, error) {
	client := s.HTTPClient()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.ipify.org", nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	buf := make([]byte, 64)
	n, _ := resp.Body.Read(buf)
	return string(buf[:n]), nil
}

// RotateCircuit signals Tor to use a new circuit.
// This requires the Tor control port (default 9051).
func (s *Smoker) RotateCircuit() error {
	if !s.IsEnabled() {
		return nil
	}

	// Connect to Tor control port
	conn, err := net.Dial("tcp", "127.0.0.1:9051")
	if err != nil {
		return fmt.Errorf("failed to connect to Tor control port: %w", err)
	}
	defer conn.Close()

	// Send SIGNAL NEWNYM command
	conn.Write([]byte("AUTHENTICATE\r\n"))
	conn.Write([]byte("SIGNAL NEWNYM\r\n"))
	conn.Write([]byte("QUIT\r\n"))

	return nil
}
