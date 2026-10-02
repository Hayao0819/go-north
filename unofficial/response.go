package unofficial

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/Hayao0819/go-north/internal/transport"
)

// ErrResponseTooLarge is returned before decoding an oversized response.
var ErrResponseTooLarge = errors.New("north unofficial: response body is too large")

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
