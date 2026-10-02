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

	"github.com/Hayao0819/go-north/internal/transport"
)

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
