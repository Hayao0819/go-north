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

	"github.com/Hayao0819/go-north/internal/transport"
)

type requestContextKey uint8

const idempotencyKey requestContextKey = iota

type idempotencyValue struct {
	key string
}

// WithIdempotencyKey adds key to write requests made with ctx. north keeps an
// idempotency result for 24 hours.
func WithIdempotencyKey(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, idempotencyKey, idempotencyValue{key: key})
}

func contextHeader(ctx context.Context) (http.Header, error) {
	header := make(http.Header)
	value, present := ctx.Value(idempotencyKey).(idempotencyValue)
	if !present {
		return header, nil
	}
	key := value.key
	if key == "" {
		return nil, errors.New("north: idempotency key must not be empty")
	}
	if len(key) > 128 {
		return nil, errors.New("north: idempotency key must be at most 128 bytes")
	}
	for _, char := range []byte(key) {
		if char < 0x21 || char > 0x7e {
			return nil, errors.New("north: idempotency key must contain visible ASCII only")
		}
	}
	header.Set("Idempotency-Key", key)

	return header, nil
}

type envelope[T any] struct {
	Data T `json:"data"`
}

func doData[T any](ctx context.Context, c *Client, method, endpoint string, query url.Values, body io.Reader, contentType string) (T, *Response, error) {
	return doDataWithHeader[T](ctx, c, method, endpoint, query, body, contentType, nil)
}

func doDataWithHeader[T any](ctx context.Context, c *Client, method, endpoint string, query url.Values, body io.Reader, contentType string, header http.Header) (T, *Response, error) {
	var out envelope[T]
	response, err := c.doWithHeader(ctx, method, endpoint, query, body, contentType, header, &out)
	if err != nil {
		var zero T
		return zero, response, err
	}

	return out.Data, response, nil
}

type dataOrValue[T any] struct {
	Value T
}

func (v *dataOrValue[T]) UnmarshalJSON(data []byte) error {
	var wrapped struct {
		Data json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return err
	}
	if len(wrapped.Data) != 0 && string(wrapped.Data) != "null" {
		return json.Unmarshal(wrapped.Data, &v.Value)
	}

	return json.Unmarshal(data, &v.Value)
}

func doDataOrValue[T any](ctx context.Context, c *Client, method, endpoint string, query url.Values, body io.Reader, contentType string, header http.Header) (T, *Response, error) {
	var out dataOrValue[T]
	response, err := c.doWithHeader(ctx, method, endpoint, query, body, contentType, header, &out)
	if err != nil {
		var zero T
		return zero, response, err
	}

	return out.Value, response, nil
}

func jsonRequest(value any) (io.Reader, string, error) {
	body, err := transport.JSONBody(value)
	if err != nil {
		return nil, "", fmt.Errorf("north: %w", err)
	}

	return body, "application/json", nil
}

func doOK(ctx context.Context, c *Client, method, endpoint string, query url.Values, body io.Reader, contentType string) (bool, *Response, error) {
	data, response, err := doData[struct {
		OK bool `json:"ok"`
	}](ctx, c, method, endpoint, query, body, contentType)

	return data.OK, response, err
}

func (c *Client) doWithHeader(ctx context.Context, method, endpoint string, query url.Values, body io.Reader, contentType string, header http.Header, out any) (*Response, error) {
	requestHeader, err := contextHeader(ctx)
	if err != nil {
		return nil, err
	}
	for name, values := range header {
		requestHeader[name] = append([]string(nil), values...)
	}

	result, err := c.transport.DoWithHeader(ctx, method, endpoint, query, body, contentType, requestHeader)
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
