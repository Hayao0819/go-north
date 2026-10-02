// Package clientconfig contains transport settings shared by API clients.
package clientconfig

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Hayao0819/go-north/internal/transport"
)

type Config struct {
	BaseURL    string
	HTTPClient *http.Client
	UserAgent  string
}

func New(baseURL, userAgent string) Config {
	return Config{
		BaseURL:    baseURL,
		HTTPClient: transport.NewHTTPClient(),
		UserAgent:  userAgent,
	}
}

func (c *Config) SetBaseURL(rawURL string) error {
	baseURL, err := transport.NormalizeBaseURL(rawURL)
	if err != nil {
		return err
	}

	c.BaseURL = baseURL
	return nil
}

func (c *Config) SetHTTPClient(client *http.Client) error {
	if client == nil {
		return errors.New("HTTP client must not be nil")
	}

	c.HTTPClient = client
	return nil
}

func (c *Config) SetUserAgent(userAgent string) error {
	if strings.ContainsAny(userAgent, "\r\n") {
		return errors.New("user agent must not contain a newline")
	}

	c.UserAgent = userAgent
	return nil
}
