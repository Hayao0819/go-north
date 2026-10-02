package unofficial

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/Hayao0819/go-north/internal/transport"
)

const (
	// DefaultBaseURL is the north web application origin.
	DefaultBaseURL = "https://north.rip"

	defaultUserAgent = "go-north/unofficial"
)

// ErrSessionCookieRequired is returned when no web session was supplied.
var ErrSessionCookieRequired = errors.New("north unofficial: session cookie is required")

// Client calls the private API used by the north web application.
type Client struct {
	transport *transport.Transport
	baseURL   *url.URL
}

// NewClient creates a client using the value of a browser Cookie header.
func NewClient(sessionCookie string, options ...Option) (*Client, error) {
	if hasControl(sessionCookie) {
		return nil, errors.New("north unofficial: session cookie contains a control character")
	}
	sessionCookie = strings.TrimSpace(sessionCookie)
	if sessionCookie == "" {
		return nil, ErrSessionCookieRequired
	}

	config, baseURL, err := configure(options)
	if err != nil {
		return nil, err
	}
	header := webHeaders(baseURL)
	header.Set("Cookie", sessionCookie)

	return configuredClient(config, baseURL, header), nil
}

// NewPublicClient creates a client without a web session. It can only call
// routes that north exposes without authentication, such as Trends.
func NewPublicClient(options ...Option) (*Client, error) {
	config, baseURL, err := configure(options)
	if err != nil {
		return nil, err
	}

	return configuredClient(config, baseURL, webHeaders(baseURL)), nil
}

// CookieHeader returns the cookies that the client sends to north.rip.
func (c *Client) CookieHeader() string {
	if c == nil || c.transport == nil {
		return ""
	}
	if header := c.transport.Header.Get("Cookie"); header != "" {
		return header
	}
	if c.transport.HTTPClient == nil || c.transport.HTTPClient.Jar == nil || c.baseURL == nil {
		return ""
	}

	request := &http.Request{Header: make(http.Header)}
	for _, cookie := range c.transport.HTTPClient.Jar.Cookies(c.baseURL) {
		request.AddCookie(cookie)
	}

	return request.Header.Get("Cookie")
}

func configuredClient(config clientConfig, baseURL *url.URL, header http.Header) *Client {
	return &Client{
		transport: &transport.Transport{
			BaseURL:    config.common.BaseURL,
			HTTPClient: config.common.HTTPClient,
			UserAgent:  config.common.UserAgent,
			Header:     header,
		},
		baseURL: apiURL(baseURL),
	}
}

func apiURL(baseURL *url.URL) *url.URL {
	target := *baseURL
	target.Path = strings.TrimRight(target.Path, "/") + "/api/"

	return &target
}

func webHeaders(baseURL *url.URL) http.Header {
	origin := baseURL.Scheme + "://" + baseURL.Host
	header := make(http.Header)
	header.Set("Origin", origin)
	header.Set("Referer", origin+"/")

	return header
}

func hasControl(value string) bool {
	for index := 0; index < len(value); index++ {
		if value[index] < 0x20 || value[index] == 0x7f {
			return true
		}
	}

	return false
}
