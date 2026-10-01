package north

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/Hayao0819/go-north/internal/transport"
)

const (
	// DefaultBaseURL is the API-only host. The north web host does not serve
	// API requests.
	DefaultBaseURL = "https://api.north.rip"
	// APIVersion is the supported north OpenAPI version.
	APIVersion = "0.47.0"

	defaultUserAgent = "go-north"
)

var (
	// ErrTokenRequired is returned when a client is created without an API key.
	ErrTokenRequired = errors.New("north: API token is required")
	// ErrResponseTooLarge is returned before decoding an unexpectedly large
	// JSON response.
	ErrResponseTooLarge = errors.New("north: response body is too large")
)

// Option configures a Client.
type Option func(*clientConfig) error

type clientConfig struct {
	baseURL    string
	httpClient *http.Client
	userAgent  string
}

// WithBaseURL sets the API endpoint. It is useful for tests and proxies.
func WithBaseURL(rawURL string) Option {
	return func(cfg *clientConfig) error {
		baseURL, err := transport.NormalizeBaseURL(rawURL)
		if err != nil {
			return fmt.Errorf("north: %w", err)
		}

		cfg.baseURL = baseURL
		return nil
	}
}

// WithHTTPClient uses client for API requests. The caller retains ownership.
func WithHTTPClient(client *http.Client) Option {
	return func(cfg *clientConfig) error {
		if client == nil {
			return errors.New("north: HTTP client must not be nil")
		}
		cfg.httpClient = client
		return nil
	}
}

// WithUserAgent sets the User-Agent request header.
func WithUserAgent(userAgent string) Option {
	return func(cfg *clientConfig) error {
		if strings.ContainsAny(userAgent, "\r\n") {
			return errors.New("north: user agent must not contain a newline")
		}
		cfg.userAgent = userAgent
		return nil
	}
}

// Client calls the north REST API. It is safe for concurrent use when its
// underlying http.Client is safe for concurrent use.
type Client struct {
	transport *transport.Transport
}

// NewClient returns a client authenticated with token.
func NewClient(token string, opts ...Option) (*Client, error) {
	if strings.ContainsAny(token, "\r\n") {
		return nil, errors.New("north: API token must not contain a newline")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrTokenRequired
	}

	cfg := clientConfig{
		baseURL:    DefaultBaseURL,
		httpClient: transport.NewHTTPClient(),
		userAgent:  defaultUserAgent,
	}

	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	header := make(http.Header)
	header.Set("Authorization", "Bearer "+token)

	return &Client{transport: &transport.Transport{
		BaseURL:    cfg.baseURL,
		HTTPClient: cfg.httpClient,
		UserAgent:  cfg.userAgent,
		Header:     header,
	}}, nil
}

// RateLimit is the request quota reported by north. Present is false when the
// response did not include rate-limit headers.
type RateLimit struct {
	Present   bool
	Limit     int
	Remaining int
	Reset     time.Time
}

// Response is transport metadata retained after the response body is closed.
type Response struct {
	StatusCode int
	Header     http.Header
	RateLimit  RateLimit
}

// ErrorDetail is one structured error returned by north.
type ErrorDetail struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// APIError is returned for non-2xx HTTP responses.
type APIError struct {
	StatusCode int
	Status     string
	Errors     []ErrorDetail
	Response   *Response
	Body       string
}

func (e *APIError) Error() string {
	if len(e.Errors) > 0 {
		return fmt.Sprintf("north: API error %d (%d): %s", e.StatusCode, e.Errors[0].Code, e.Errors[0].Message)
	}
	if e.Body != "" {
		return fmt.Sprintf("north: API error %d: %s", e.StatusCode, e.Body)
	}

	return fmt.Sprintf("north: API error %d: %s", e.StatusCode, e.Status)
}

// HasCode reports whether the response contains a particular north error code.
func (e *APIError) HasCode(code int) bool {
	for _, item := range e.Errors {
		if item.Code == code {
			return true
		}
	}

	return false
}

type envelope[T any] struct {
	Data T `json:"data"`
}

func doData[T any](ctx context.Context, c *Client, method, endpoint string, query url.Values, body io.Reader, contentType string) (T, *Response, error) {
	var out envelope[T]
	resp, err := c.do(ctx, method, endpoint, query, body, contentType, &out)
	if err != nil {
		var zero T
		return zero, resp, err
	}
	return out.Data, resp, nil
}

func jsonRequest(v any) (io.Reader, string, error) {
	body, err := transport.JSONBody(v)
	if err != nil {
		return nil, "", fmt.Errorf("north: %w", err)
	}

	return body, "application/json", nil
}

func (c *Client) do(ctx context.Context, method, endpoint string, query url.Values, body io.Reader, contentType string, out any) (*Response, error) {
	result, err := c.transport.Do(ctx, method, endpoint, query, body, contentType)
	response := newResponse(result)
	if err != nil {
		if errors.Is(err, transport.ErrResponseTooLarge) {
			return response, ErrResponseTooLarge
		}

		return response, fmt.Errorf("north: %w", err)
	}

	if result.StatusCode < http.StatusOK || result.StatusCode >= http.StatusMultipleChoices {
		return response, decodeAPIError(result, response)
	}
	if out == nil || len(bytes.TrimSpace(result.Body)) == 0 {
		return response, nil
	}
	if err := json.Unmarshal(result.Body, out); err != nil {
		return response, fmt.Errorf("north: decode response: %w", err)
	}

	return response, nil
}

func newResponse(result transport.Result) *Response {
	if result.StatusCode == 0 {
		return nil
	}

	return &Response{
		StatusCode: result.StatusCode,
		Header:     result.Header,
		RateLimit:  parseRateLimit(result.Header),
	}
}

func parseRateLimit(h http.Header) RateLimit {
	limitHeader := h.Get("X-Rate-Limit-Limit")
	remainingHeader := h.Get("X-Rate-Limit-Remaining")
	resetHeader := h.Get("X-Rate-Limit-Reset")
	if limitHeader == "" && remainingHeader == "" && resetHeader == "" {
		return RateLimit{}
	}

	limit, _ := strconv.Atoi(limitHeader)
	remaining, _ := strconv.Atoi(remainingHeader)
	reset, _ := strconv.ParseInt(resetHeader, 10, 64)

	rate := RateLimit{Present: true, Limit: limit, Remaining: remaining}
	if reset > 0 {
		rate.Reset = time.Unix(reset, 0)
	}
	return rate
}

func decodeAPIError(result transport.Result, response *Response) error {
	payload := struct {
		Errors []ErrorDetail `json:"errors"`
	}{}
	_ = json.Unmarshal(result.Body, &payload)

	text := strings.TrimSpace(string(result.Body))
	if len(text) > 1024 {
		text = text[:1024] + "…"
	}

	return &APIError{
		StatusCode: result.StatusCode,
		Status:     result.Status,
		Errors:     payload.Errors,
		Response:   response,
		Body:       text,
	}
}
