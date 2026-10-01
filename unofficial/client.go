package unofficial

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

var (
	// ErrSessionCookieRequired is returned when no web session was supplied.
	ErrSessionCookieRequired = errors.New("north unofficial: session cookie is required")
	// ErrResponseTooLarge is returned before decoding an oversized response.
	ErrResponseTooLarge = errors.New("north unofficial: response body is too large")
)

// Option configures a Client.
type Option func(*clientConfig) error

type clientConfig struct {
	baseURL    string
	httpClient *http.Client
	userAgent  string
}

// WithBaseURL changes the web application origin. It is mainly useful for tests.
func WithBaseURL(rawURL string) Option {
	return func(cfg *clientConfig) error {
		baseURL, err := transport.NormalizeBaseURL(rawURL)
		if err != nil {
			return fmt.Errorf("north unofficial: %w", err)
		}

		cfg.baseURL = baseURL

		return nil
	}
}

// WithHTTPClient uses client for requests. The caller retains ownership.
func WithHTTPClient(client *http.Client) Option {
	return func(cfg *clientConfig) error {
		if client == nil {
			return errors.New("north unofficial: HTTP client must not be nil")
		}
		cfg.httpClient = client

		return nil
	}
}

// WithUserAgent sets the User-Agent request header.
func WithUserAgent(userAgent string) Option {
	return func(cfg *clientConfig) error {
		if strings.ContainsAny(userAgent, "\r\n") {
			return errors.New("north unofficial: user agent must not contain a newline")
		}
		cfg.userAgent = userAgent

		return nil
	}
}

// Client calls the private API used by the north web application.
type Client struct {
	transport *transport.Transport
	baseURL   *url.URL
}

// NewClient creates a client using the value of a browser Cookie header.
func NewClient(sessionCookie string, opts ...Option) (*Client, error) {
	if hasControl(sessionCookie) {
		return nil, errors.New("north unofficial: session cookie contains a control character")
	}
	sessionCookie = strings.TrimSpace(sessionCookie)
	if sessionCookie == "" {
		return nil, ErrSessionCookieRequired
	}

	cfg, baseURL, err := configure(opts)
	if err != nil {
		return nil, err
	}
	header := webHeaders(baseURL)
	header.Set("Cookie", sessionCookie)

	return configuredClient(cfg, baseURL, header), nil
}

// NewPublicClient creates a client without a web session. It can only call
// routes that north exposes without authentication, such as Trends.
func NewPublicClient(opts ...Option) (*Client, error) {
	cfg, baseURL, err := configure(opts)
	if err != nil {
		return nil, err
	}

	return configuredClient(cfg, baseURL, webHeaders(baseURL)), nil
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

func configure(opts []Option) (clientConfig, *url.URL, error) {
	cfg := clientConfig{
		baseURL:    DefaultBaseURL,
		httpClient: transport.NewHTTPClient(),
		userAgent:  defaultUserAgent,
	}
	for _, option := range opts {
		if option == nil {
			continue
		}
		if err := option(&cfg); err != nil {
			return clientConfig{}, nil, err
		}
	}

	baseURL, err := url.Parse(cfg.baseURL)
	if err != nil {
		return clientConfig{}, nil, fmt.Errorf("north unofficial: parse configured base URL: %w", err)
	}

	return cfg, baseURL, nil
}

func configuredClient(cfg clientConfig, baseURL *url.URL, header http.Header) *Client {
	return &Client{
		transport: &transport.Transport{
			BaseURL:    cfg.baseURL,
			HTTPClient: cfg.httpClient,
			UserAgent:  cfg.userAgent,
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

// Response contains transport metadata from the web API.
type Response struct {
	StatusCode int
	Header     http.Header
}

// APIError is returned for a non-2xx web API response.
type APIError struct {
	StatusCode int
	Status     string
	Code       string
	Message    string
	Fields     map[string]json.RawMessage
	Response   *Response
	Body       string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		if e.Code != "" {
			return fmt.Sprintf("north unofficial: API error %d (%s): %s", e.StatusCode, e.Code, e.Message)
		}

		return fmt.Sprintf("north unofficial: API error %d: %s", e.StatusCode, e.Message)
	}

	return fmt.Sprintf("north unofficial: API error %d: %s", e.StatusCode, e.Status)
}

func doJSON[T any](ctx context.Context, client *Client, method, endpoint string, query url.Values, input any) (T, *Response, error) {
	body, err := encodeRequest(input)
	if err != nil {
		var zero T

		return zero, nil, err
	}

	var output T
	response, err := client.do(ctx, method, endpoint, query, body, &output)
	if err != nil {
		var zero T

		return zero, response, err
	}

	return output, response, nil
}

func doEmpty(ctx context.Context, client *Client, method, endpoint string, query url.Values, input any) (*Response, error) {
	body, err := encodeRequest(input)
	if err != nil {
		return nil, err
	}

	return client.do(ctx, method, endpoint, query, body, nil)
}

func encodeRequest(input any) (io.Reader, error) {
	body, err := transport.JSONBody(input)
	if err != nil {
		return nil, fmt.Errorf("north unofficial: %w", err)
	}

	return body, nil
}

func (c *Client) do(ctx context.Context, method, endpoint string, query url.Values, body io.Reader, output any) (*Response, error) {
	contentType := ""
	if body != nil {
		contentType = "application/json"
	}

	result, err := c.transport.Do(ctx, method, endpoint, query, body, contentType)
	response := newResponse(result)
	if err != nil {
		if errors.Is(err, transport.ErrResponseTooLarge) {
			return response, ErrResponseTooLarge
		}

		return response, fmt.Errorf("north unofficial: %w", err)
	}
	if result.StatusCode < http.StatusOK || result.StatusCode >= http.StatusMultipleChoices {
		return response, decodeAPIError(result, response)
	}
	if output == nil || len(bytes.TrimSpace(result.Body)) == 0 {
		return response, nil
	}
	if err := json.Unmarshal(result.Body, output); err != nil {
		return response, fmt.Errorf("north unofficial: decode response: %w", err)
	}

	return response, nil
}

func newResponse(result transport.Result) *Response {
	if result.StatusCode == 0 {
		return nil
	}

	return &Response{StatusCode: result.StatusCode, Header: result.Header}
}

func decodeAPIError(result transport.Result, response *Response) error {
	payload := struct {
		Code    string                     `json:"error"`
		Message string                     `json:"message"`
		Fields  map[string]json.RawMessage `json:"fields"`
	}{}
	_ = json.Unmarshal(result.Body, &payload)

	text := strings.TrimSpace(string(result.Body))
	if len(text) > 1024 {
		text = text[:1024] + "…"
	}

	return &APIError{
		StatusCode: result.StatusCode,
		Status:     result.Status,
		Code:       payload.Code,
		Message:    payload.Message,
		Fields:     payload.Fields,
		Response:   response,
		Body:       text,
	}
}

func hasControl(value string) bool {
	for index := 0; index < len(value); index++ {
		if value[index] < 0x20 || value[index] == 0x7f {
			return true
		}
	}

	return false
}
