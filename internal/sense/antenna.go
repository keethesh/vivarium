package sense

import (
	"context"
	"net/http"
	"strings"

	"vivarium/internal/common"
)

// Antenna detects WAFs and server technologies.
type Antenna struct {
	client *http.Client
}

// NewAntenna creates a new Antenna detector.
func NewAntenna() *Antenna {
	return &Antenna{
		client: common.DefaultHTTPClient(),
	}
}

// TechInfo contains detected technology information.
type TechInfo struct {
	Server    string
	PoweredBy string
	WAF       string
	Cookies   []string
}

// Detect scans the target URL for WAFs and technology signatures.
func (a *Antenna) Detect(ctx context.Context, targetURL string) (*TechInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", targetURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", common.RandomUserAgent())

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	info := &TechInfo{
		Server:    resp.Header.Get("Server"),
		PoweredBy: resp.Header.Get("X-Powered-By"),
	}

	// Detect WAFs based on headers and cookies/body
	info.WAF = a.detectWAF(resp)

	for _, cookie := range resp.Cookies() {
		info.Cookies = append(info.Cookies, cookie.Name)
	}

	return info, nil
}

// detectWAF checks headers and response characteristics for known WAF signatures.
func (a *Antenna) detectWAF(resp *http.Response) string {
	headers := resp.Header

	// Cloudflare
	if headers.Get("Server") == "cloudflare" || headers.Get("CF-RAY") != "" {
		return "Cloudflare"
	}

	// AWS WAF
	if headers.Get("X-Amz-Cf-Id") != "" {
		return "AWS CloudFront"
	}

	// Akamai
	if headers.Get("X-Akamai-Transformed") != "" {
		return "Akamai"
	}

	// Imperva Incapsula
	if headers.Get("X-Iinfo") != "" || headers.Get("X-CDN") == "Incapsula" {
		return "Imperva Incapsula"
	}

	// F5 BIG-IP
	for _, cookie := range resp.Cookies() {
		if strings.HasPrefix(cookie.Name, "BIGipServer") {
			return "F5 BIG-IP"
		}
	}

	return ""
}
