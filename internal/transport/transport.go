package transport

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
	"time"
)

const maxBodySize = 32 << 20

// ErrResponseTooLarge is returned when a response exceeds the read limit.
var ErrResponseTooLarge = errors.New("response body is too large")

// Transport sends requests with a shared base URL and headers.
type Transport struct {
	BaseURL    string
	HTTPClient *http.Client
	UserAgent  string
	Header     http.Header
}

// Result is the part of an HTTP response needed by the API packages.
type Result struct {
	StatusCode int
	Status     string
	Header     http.Header
	Body       []byte
}

// NormalizeBaseURL validates a base URL and removes its trailing slash.
func NormalizeBaseURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("parse base URL: %w", err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", errors.New("base URL must be an absolute HTTP URL")
	}
	if u.ForceQuery || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("base URL must not contain a query or fragment")
	}

	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath = ""

	return u.String(), nil
}

// NewHTTPClient returns the default client used for regular API requests.
func NewHTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second}
}

// NewStreamingHTTPClient returns a client without a whole-request timeout.
// Streaming requests must be bounded by their context instead.
func NewStreamingHTTPClient() *http.Client {
	return &http.Client{}
}

// JSONBody encodes a request body. A nil value produces no body.
func JSONBody(value any) (io.Reader, error) {
	if value == nil {
		return nil, nil
	}

	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(value); err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	return &body, nil
}

// Do sends one request and reads its response body.
func (c *Transport) Do(ctx context.Context, method, endpoint string, query url.Values, body io.Reader, contentType string) (Result, error) {
	var result Result
	response, err := c.Open(ctx, method, endpoint, query, body, contentType, "application/json")
	if err != nil {
		return result, err
	}
	defer response.Body.Close()

	return ReadResponse(response)
}

// Open sends one request and leaves its response body open for the caller.
func (c *Transport) Open(
	ctx context.Context,
	method string,
	endpoint string,
	query url.Values,
	body io.Reader,
	contentType string,
	accept string,
) (*http.Response, error) {
	u, err := url.Parse(c.BaseURL + endpoint)
	if err != nil {
		return nil, fmt.Errorf("build request URL: %w", err)
	}
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}

	request, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if accept != "" {
		request.Header.Set("Accept", accept)
	}
	for name, values := range c.Header {
		request.Header[name] = append([]string(nil), values...)
	}
	if c.UserAgent != "" {
		request.Header.Set("User-Agent", c.UserAgent)
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}

	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	return response, nil
}

// ReadResponse reads a bounded response body. The caller must close it.
func ReadResponse(response *http.Response) (Result, error) {
	var result Result
	result.StatusCode = response.StatusCode
	result.Status = response.Status
	result.Header = response.Header.Clone()
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBodySize+1))
	if err != nil {
		return result, fmt.Errorf("read response: %w", err)
	}
	result.Body = body
	if len(result.Body) > maxBodySize {
		return result, ErrResponseTooLarge
	}

	return result, nil
}
