package north

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Hayao0819/go-north/internal/transport"
)

const (
	// DefaultBaseURL is the API-only host. The north web host does not serve
	// API requests.
	DefaultBaseURL = "https://api.north.rip"
	// APIVersion is the supported north OpenAPI version.
	APIVersion = "0.56.1"

	defaultUserAgent = "go-north"
)

// ErrTokenRequired is returned when a client is created without a token.
var ErrTokenRequired = errors.New("north: API token is required")

// Client calls the north REST API. It is safe for concurrent use when its
// underlying http.Client is safe for concurrent use.
type Client struct {
	transport          *transport.Transport
	streamingTransport *transport.Transport
	tokenKind          TokenKind
}

// NewClient returns a client authenticated with a personal token or legacy API
// key.
func NewClient(token string, opts ...Option) (*Client, error) {
	var err error
	token, err = normalizeToken(token)
	if err != nil {
		return nil, err
	}

	cfg, err := configureClient(opts)
	if err != nil {
		return nil, err
	}

	header := make(http.Header)
	header.Set("Authorization", "Bearer "+token)

	return newClient(cfg, header, nil, DetectTokenKind(token)), nil
}

func normalizeToken(token string) (string, error) {
	if strings.ContainsAny(token, "\r\n") {
		return "", errors.New("north: API token must not contain a newline")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return "", ErrTokenRequired
	}

	return token, nil
}

func newClient(cfg clientConfig, header http.Header, bearerToken func() (string, error), tokenKind TokenKind) *Client {
	return &Client{
		transport: &transport.Transport{
			BaseURL:     cfg.common.BaseURL,
			HTTPClient:  cfg.common.HTTPClient,
			UserAgent:   cfg.common.UserAgent,
			Header:      header,
			BearerToken: bearerToken,
		},
		streamingTransport: &transport.Transport{
			BaseURL:     cfg.common.BaseURL,
			HTTPClient:  cfg.streamingClient,
			UserAgent:   cfg.common.UserAgent,
			Header:      header,
			BearerToken: bearerToken,
		},
		tokenKind: tokenKind,
	}
}
