package oauth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/oauth2"
)

// Revoke removes the grant associated with token. endpoint is normally
// RevokeURL.
func Revoke(ctx context.Context, client *http.Client, endpoint, clientID, token string) error {
	if client == nil {
		client = http.DefaultClient
	}
	values := url.Values{
		"client_id": {clientID},
		"token":     {token},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return fmt.Errorf("north oauth: build revoke request: %w", err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("north oauth: revoke token: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("north oauth: read revoke response: %w", err)
	}
	errorResponse := &oauth2.RetrieveError{Response: response, Body: body}
	if len(bytes.TrimSpace(body)) != 0 {
		var payload struct {
			Code        string `json:"error"`
			Description string `json:"error_description"`
			URI         string `json:"error_uri"`
		}
		if err := json.Unmarshal(body, &payload); err == nil {
			errorResponse.ErrorCode = payload.Code
			errorResponse.ErrorDescription = payload.Description
			errorResponse.ErrorURI = payload.URI
		}
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices || errorResponse.ErrorCode != "" {
		return errorResponse
	}

	return nil
}
