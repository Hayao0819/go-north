package north

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Hayao0819/go-north/internal/transport"
)

// ErrResponseTooLarge is returned before decoding an unexpectedly large JSON
// response.
var ErrResponseTooLarge = errors.New("north: response body is too large")

// RateLimit is the request quota reported by north. Present is false when the
// response did not include rate-limit headers.
type RateLimit struct {
	Present   bool
	Limit     int
	Remaining int
	Reset     time.Time
	Scope     string
}

// Response is transport metadata retained after the response body is closed.
type Response struct {
	StatusCode int
	Header     http.Header
	RateLimit  RateLimit
	RetryAfter time.Duration
}

// ErrorDetail is one structured error returned by north.
type ErrorDetail struct {
	Code           int      `json:"code"`
	Message        string   `json:"message"`
	RequiredScopes []string `json:"required_scopes"`
	LimitScope     string   `json:"limit_scope"`
}

// APIError is returned for non-2xx HTTP responses.
type APIError struct {
	StatusCode int
	Status     string
	Errors     []ErrorDetail
	RequestID  string
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

func newResponse(result transport.Result) *Response {
	if result.StatusCode == 0 {
		return nil
	}

	return &Response{
		StatusCode: result.StatusCode,
		Header:     result.Header,
		RateLimit:  parseRateLimit(result.Header),
		RetryAfter: transport.RetryAfter(result.Header),
	}
}

func parseRateLimit(header http.Header) RateLimit {
	limitHeader := header.Get("X-Rate-Limit-Limit")
	remainingHeader := header.Get("X-Rate-Limit-Remaining")
	resetHeader := header.Get("X-Rate-Limit-Reset")
	if limitHeader == "" && remainingHeader == "" && resetHeader == "" {
		return RateLimit{}
	}

	limit, _ := strconv.Atoi(limitHeader)
	remaining, _ := strconv.Atoi(remainingHeader)
	reset, _ := strconv.ParseInt(resetHeader, 10, 64)
	rate := RateLimit{
		Present:   true,
		Limit:     limit,
		Remaining: remaining,
		Scope:     header.Get("X-North-Rate-Limit-Scope"),
	}
	if reset > 0 {
		rate.Reset = time.Unix(reset, 0)
	}

	return rate
}

func decodeAPIError(result transport.Result, response *Response) error {
	payload := struct {
		Errors    []ErrorDetail `json:"errors"`
		RequestID string        `json:"request_id"`
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
		RequestID:  payload.RequestID,
		Response:   response,
		Body:       text,
	}
}
