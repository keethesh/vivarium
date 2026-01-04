package common

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

var (
	globalProxy     string
	globalProxyOnce sync.Once
	globalDialer    proxy.Dialer
)

// SetGlobalProxy sets the global proxy URL for all HTTP clients.
func SetGlobalProxy(proxyURL string) error {
	if proxyURL == "" {
		return nil
	}

	u, err := url.Parse(proxyURL)
	if err != nil {
		return err
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
		return err
	}

	globalProxy = proxyURL
	globalDialer = dialer
	return nil
}

// GetGlobalProxy returns the current global proxy URL.
func GetGlobalProxy() string {
	return globalProxy
}

// IsProxyEnabled returns whether a global proxy is configured.
func IsProxyEnabled() bool {
	return globalProxy != "" && globalDialer != nil
}

// DefaultHTTPClient returns an HTTP client configured for stress testing.
// It has aggressive timeouts and connection pooling.
func DefaultHTTPClient() *http.Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 1000,
		MaxConnsPerHost:     1000,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // Skip cert verification for stress testing
		},
		DisableKeepAlives: false,
	}

	// Use proxy if configured
	if IsProxyEnabled() {
		transport.Dial = globalDialer.Dial
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Follow up to 10 redirects
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
}

// FastHTTPClient returns an HTTP client optimized for speed over reliability.
func FastHTTPClient() *http.Client {
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   5 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        2000,
		MaxIdleConnsPerHost: 2000,
		MaxConnsPerHost:     2000,
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: 5 * time.Second,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
		DisableKeepAlives:     false,
		DisableCompression:    true, // Skip decompression overhead
		ResponseHeaderTimeout: 10 * time.Second,
	}

	// Use proxy if configured
	if IsProxyEnabled() {
		transport.Dial = globalDialer.Dial
	}

	return &http.Client{
		Transport: transport,
		Timeout:   15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Don't follow redirects - we just want to hit the server
			return http.ErrUseLastResponse
		},
	}
}
