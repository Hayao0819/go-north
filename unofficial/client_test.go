package unofficial

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

type endpointRequest struct {
	method string
	path   string
	query  url.Values
	body   map[string]any
}

type endpointTest struct {
	name   string
	method string
	path   string
	query  url.Values
	body   map[string]any
	call   func(context.Context, *Client) error
}

func testEndpoints(t *testing.T, tests []endpointTest) {
	t.Helper()

	seen := make(chan endpointRequest, 1)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		body := make(map[string]any)
		if request.Body != nil {
			data, err := io.ReadAll(request.Body)
			if err != nil {
				t.Errorf("read request: %v", err)
			}
			if len(data) > 0 {
				if err := json.Unmarshal(data, &body); err != nil {
					t.Errorf("decode request: %v", err)
				}
			}
		}
		seen <- endpointRequest{
			method: request.Method,
			path:   request.URL.EscapedPath(),
			query:  request.URL.Query(),
			body:   body,
		}
		writeJSON(t, writer, http.StatusOK, `{}`)
	}))
	t.Cleanup(server.Close)

	client, err := NewClient("session=secret", WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.call(context.Background(), client); err != nil {
				t.Fatal(err)
			}
			request := <-seen
			if request.method != test.method || request.path != test.path || request.query.Encode() != test.query.Encode() {
				t.Errorf("request = %s %s?%s, want %s %s?%s", request.method, request.path, request.query.Encode(), test.method, test.path, test.query.Encode())
			}
			if test.body == nil {
				if len(request.body) != 0 {
					t.Errorf("body = %#v, want no body", request.body)
				}
			} else if !reflect.DeepEqual(request.body, test.body) {
				t.Errorf("body = %#v, want %#v", request.body, test.body)
			}
		})
	}
}

func TestClientSendsWebSessionHeaders(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Cookie"); got != "session=secret" {
			t.Errorf("Cookie = %q", got)
		}
		if got := request.Header.Get("Origin"); got != "http://"+request.Host {
			t.Errorf("Origin = %q", got)
		}
		if got := request.Header.Get("Referer"); got != "http://"+request.Host+"/" {
			t.Errorf("Referer = %q", got)
		}
		writeJSON(t, writer, http.StatusOK, `{"count":4}`)
	}))
	t.Cleanup(server.Close)

	client, err := NewClient("session=secret", WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	count, response, err := client.NotificationUnreadCount(context.Background())
	if err != nil || count != 4 || response.StatusCode != http.StatusOK {
		t.Fatalf("NotificationUnreadCount = %d, %#v, %v", count, response, err)
	}
}

func TestClientRejectsMissingOrUnsafeCookie(t *testing.T) {
	t.Parallel()

	if _, err := NewClient("  "); !errors.Is(err, ErrSessionCookieRequired) {
		t.Fatalf("blank cookie error = %v", err)
	}
	if _, err := NewClient("session=value\r\nX-Test: value"); err == nil {
		t.Fatal("cookie containing a newline was accepted")
	}
}

func TestClientDecodesWebAPIError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(t, writer, http.StatusForbidden, `{"error":"forbidden","message":"not allowed","fields":{"text":"invalid"}}`)
	}))
	t.Cleanup(server.Close)

	client, err := NewClient("session=secret", WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = client.Post(context.Background(), "1")
	var apiError *APIError
	if !errors.As(err, &apiError) || apiError.Code != "forbidden" || apiError.Message != "not allowed" {
		t.Fatalf("error = %#v", err)
	}
}

func writeJSON(t *testing.T, writer http.ResponseWriter, status int, body string) {
	t.Helper()
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	if _, err := writer.Write([]byte(body)); err != nil {
		t.Errorf("write response: %v", err)
	}
}
