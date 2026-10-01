package north

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	client, err := NewClient(
		"nth_live_test",
		WithBaseURL(server.URL+"/api/"),
		WithHTTPClient(server.Client()),
		WithUserAgent("go-north-test"),
	)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	return client
}

func writeJSON(t *testing.T, writer http.ResponseWriter, status int, body string) {
	t.Helper()

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	if _, err := writer.Write([]byte(body)); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

func TestNewClientValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		token   string
		options []Option
		want    string
	}{
		{name: "empty token", want: ErrTokenRequired.Error()},
		{name: "blank token", token: "  ", want: ErrTokenRequired.Error()},
		{name: "newline token", token: "nth_live_x\n", want: "newline"},
		{name: "relative URL", token: "token", options: []Option{WithBaseURL("api.example")}, want: "absolute HTTP"},
		{name: "query in URL", token: "token", options: []Option{WithBaseURL("https://api.example/?x=1")}, want: "query or fragment"},
		{name: "empty query in URL", token: "token", options: []Option{WithBaseURL("https://api.example/?")}, want: "query or fragment"},
		{name: "nil HTTP client", token: "token", options: []Option{WithHTTPClient(nil)}, want: "must not be nil"},
		{name: "newline user agent", token: "token", options: []Option{WithUserAgent("client\nbad")}, want: "newline"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, err := NewClient(test.token, test.options...)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("NewClient error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestRequestHeadersBasePathAndRateLimit(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/api/2/users/me" {
			t.Errorf("path = %q", request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer nth_live_test" {
			t.Errorf("Authorization = %q", got)
		}
		if got := request.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q", got)
		}
		if got := request.Header.Get("User-Agent"); got != "go-north-test" {
			t.Errorf("User-Agent = %q", got)
		}

		writer.Header().Set("X-Rate-Limit-Limit", "75")
		writer.Header().Set("X-Rate-Limit-Remaining", "74")
		writer.Header().Set("X-Rate-Limit-Reset", "1772500000")
		writeJSON(t, writer, http.StatusOK, `{"data":{"id":"1","handle":"north","name":"North"}}`)
	})

	user, response, err := client.Me(context.Background())
	if err != nil {
		t.Fatalf("Me: %v", err)
	}
	if user.Handle != "north" {
		t.Errorf("handle = %q", user.Handle)
	}
	if response.StatusCode != http.StatusOK || !response.RateLimit.Present {
		t.Fatalf("response = %#v", response)
	}
	if response.RateLimit.Limit != 75 || response.RateLimit.Remaining != 74 {
		t.Errorf("rate limit = %#v", response.RateLimit)
	}
	wantReset := time.Unix(1772500000, 0)
	if !response.RateLimit.Reset.Equal(wantReset) {
		t.Errorf("reset = %v, want %v", response.RateLimit.Reset, wantReset)
	}
}

func TestStructuredAPIError(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("X-Rate-Limit-Remaining", "0")
		writeJSON(t, writer, http.StatusTooManyRequests, `{"errors":[{"code":88,"message":"Rate limit exceeded"}]}`)
	})

	_, response, err := client.Me(context.Background())
	if err == nil {
		t.Fatal("Me succeeded")
	}
	if response == nil || response.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("response = %#v", response)
	}

	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("error type = %T", err)
	}
	if !apiError.HasCode(88) || apiError.HasCode(34) {
		t.Errorf("HasCode on %#v", apiError.Errors)
	}
	if strings.Contains(err.Error(), "nth_live_test") {
		t.Errorf("error leaked API token: %v", err)
	}
}

func TestMalformedSuccessRetainsResponse(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(t, writer, http.StatusOK, `{not json`)
	})

	_, response, err := client.Me(context.Background())
	if err == nil || !strings.Contains(err.Error(), "decode response") {
		t.Fatalf("error = %v", err)
	}
	if response == nil || response.StatusCode != http.StatusOK {
		t.Fatalf("response = %#v", response)
	}
}
