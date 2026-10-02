package north

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
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

func TestTokenKind(t *testing.T) {
	t.Parallel()

	tests := []struct {
		token string
		kind  TokenKind
	}{
		{"nth_live_old", TokenLegacy},
		{"nth_oat_new", TokenScoped},
		{" test-token ", TokenUnknown},
	}
	for _, test := range tests {
		client, err := NewClient(test.token)
		if err != nil {
			t.Fatal(err)
		}
		if client.TokenKind() != test.kind || DetectTokenKind(test.token) != test.kind {
			t.Errorf("token %q: kind = %v", test.token, client.TokenKind())
		}
	}
}

func TestNewClientWithTokenSource(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer nth_oat_refreshed" {
			t.Errorf("Authorization = %q", request.Header.Get("Authorization"))
		}
		writeJSON(t, writer, http.StatusOK, `{"data":{"id":"1","handle":"north","name":"North"}}`)
	}))
	defer server.Close()

	client, err := NewClientWithTokenSource(
		oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "nth_oat_refreshed", TokenType: "Bearer"}),
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
	)
	if err != nil {
		t.Fatal(err)
	}
	if client.TokenKind() != TokenScoped {
		t.Fatalf("TokenKind = %v", client.TokenKind())
	}
	if _, _, err := client.Me(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestNewClientWithTokenSourceValidation(t *testing.T) {
	t.Parallel()

	if _, err := NewClientWithTokenSource(nil); !errors.Is(err, ErrTokenSourceRequired) {
		t.Fatalf("nil token source error = %v", err)
	}
	client, err := NewClientWithTokenSource(oauth2.StaticTokenSource(&oauth2.Token{}))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := client.Me(context.Background()); !errors.Is(err, ErrTokenRequired) {
		t.Fatalf("empty token error = %v", err)
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
		writer.Header().Set("X-North-Rate-Limit-Scope", "grant")
		writer.Header().Set("Retry-After", "3")
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
	if response.RateLimit.Scope != "grant" || response.RetryAfter != 3*time.Second {
		t.Errorf("response metadata = %#v", response)
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
		writeJSON(t, writer, http.StatusTooManyRequests, `{"errors":[{"code":88,"message":"Rate limit exceeded","required_scopes":["posts.read"],"limit_scope":"grant"}],"request_id":"request-1"}`)
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
	if apiError.RequestID != "request-1" || apiError.Errors[0].LimitScope != "grant" || len(apiError.Errors[0].RequiredScopes) != 1 {
		t.Errorf("API error metadata = %#v", apiError)
	}
	if strings.Contains(err.Error(), "nth_live_test") {
		t.Errorf("error leaked API token: %v", err)
	}
}

func TestIdempotencyKey(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(writer http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Idempotency-Key"); got != "create-1" {
			t.Errorf("Idempotency-Key = %q", got)
		}
		writeJSON(t, writer, http.StatusCreated, `{"data":{"id":"1","text":"hello"}}`)
	})
	ctx := WithIdempotencyKey(context.Background(), "create-1")
	if _, _, err := client.CreatePost(ctx, CreatePostRequest{Text: "hello"}); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidIdempotencyKey(t *testing.T) {
	t.Parallel()

	client, err := NewClient("token")
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"", "contains space", strings.Repeat("x", 129)} {
		ctx := WithIdempotencyKey(context.Background(), key)
		if _, _, err := client.CreatePost(ctx, CreatePostRequest{Text: "hello"}); err == nil {
			t.Errorf("CreatePost accepted idempotency key %q", key)
		}
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
