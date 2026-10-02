package unofficial

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/Hayao0819/go-north/internal/clientconfig"
)

// Option configures a Client.
type Option func(*clientConfig) error

type clientConfig struct {
	common clientconfig.Config
}

// WithBaseURL changes the web application origin. It is mainly useful for tests.
func WithBaseURL(rawURL string) Option {
	return func(config *clientConfig) error {
		if err := config.common.SetBaseURL(rawURL); err != nil {
			return fmt.Errorf("north unofficial: %w", err)
		}

		return nil
	}
}

// WithHTTPClient uses client for requests. The caller retains ownership.
func WithHTTPClient(client *http.Client) Option {
	return func(config *clientConfig) error {
		if err := config.common.SetHTTPClient(client); err != nil {
			return fmt.Errorf("north unofficial: %w", err)
		}

		return nil
	}
}

// WithUserAgent sets the User-Agent request header.
func WithUserAgent(userAgent string) Option {
	return func(config *clientConfig) error {
		if err := config.common.SetUserAgent(userAgent); err != nil {
			return fmt.Errorf("north unofficial: %w", err)
		}

		return nil
	}
}

func configure(options []Option) (clientConfig, *url.URL, error) {
	config := clientConfig{common: clientconfig.New(DefaultBaseURL, defaultUserAgent)}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&config); err != nil {
			return clientConfig{}, nil, err
		}
	}

	baseURL, err := url.Parse(config.common.BaseURL)
	if err != nil {
		return clientConfig{}, nil, fmt.Errorf("north unofficial: parse configured base URL: %w", err)
	}

	return config, baseURL, nil
}
