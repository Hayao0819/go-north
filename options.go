package north

import (
	"fmt"
	"net/http"

	"github.com/Hayao0819/go-north/internal/clientconfig"
	"github.com/Hayao0819/go-north/internal/transport"
)

// Option configures a Client.
type Option func(*clientConfig) error

type clientConfig struct {
	common          clientconfig.Config
	streamingClient *http.Client
}

// WithBaseURL sets the API endpoint. It is useful for tests and proxies.
func WithBaseURL(rawURL string) Option {
	return func(options *clientConfig) error {
		if err := options.common.SetBaseURL(rawURL); err != nil {
			return fmt.Errorf("north: %w", err)
		}

		return nil
	}
}

// WithHTTPClient uses client for API requests. The caller retains ownership.
func WithHTTPClient(client *http.Client) Option {
	return func(options *clientConfig) error {
		if err := options.common.SetHTTPClient(client); err != nil {
			return fmt.Errorf("north: %w", err)
		}
		options.streamingClient = client

		return nil
	}
}

// WithUserAgent sets the User-Agent request header.
func WithUserAgent(userAgent string) Option {
	return func(options *clientConfig) error {
		if err := options.common.SetUserAgent(userAgent); err != nil {
			return fmt.Errorf("north: %w", err)
		}

		return nil
	}
}

func configureClient(options []Option) (clientConfig, error) {
	config := clientConfig{
		common:          clientconfig.New(DefaultBaseURL, defaultUserAgent),
		streamingClient: transport.NewStreamingHTTPClient(),
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&config); err != nil {
			return clientConfig{}, err
		}
	}

	return config, nil
}
