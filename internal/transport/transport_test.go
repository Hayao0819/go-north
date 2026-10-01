package transport

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNormalizeBaseURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{name: "HTTPS", value: "https://api.example/v1///", want: "https://api.example/v1"},
		{name: "HTTP", value: "http://localhost:8080", want: "http://localhost:8080"},
		{name: "relative", value: "api.example", wantErr: true},
		{name: "scheme", value: "ftp://api.example", wantErr: true},
		{name: "query", value: "https://api.example?x=1", wantErr: true},
		{name: "empty query", value: "https://api.example?", wantErr: true},
		{name: "fragment", value: "https://api.example#fragment", wantErr: true},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := NormalizeBaseURL(test.value)
			if test.wantErr {
				if err == nil {
					t.Fatalf("NormalizeBaseURL(%q) = %q, want an error", test.value, got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != test.want {
				t.Fatalf("NormalizeBaseURL(%q) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}

func TestJSONBody(t *testing.T) {
	t.Parallel()

	body, err := JSONBody(map[string]string{"text": "hello"})
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]string
	if err := json.NewDecoder(body).Decode(&value); err != nil {
		t.Fatal(err)
	}
	if value["text"] != "hello" {
		t.Fatalf("body = %#v", value)
	}

	empty, err := JSONBody(nil)
	if err != nil || empty != nil {
		t.Fatalf("JSONBody(nil) = %#v, %v", empty, err)
	}
	if _, err := JSONBody(func() {}); err == nil {
		t.Fatal("JSONBody accepted an unsupported value")
	}
}

func TestTransportDo(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost || request.URL.Path != "/base/resource" || request.URL.Query().Get("cursor") != "next" {
			t.Errorf("request = %s %s", request.Method, request.URL.String())
		}
		if request.Header.Get("Accept") != "application/json" || request.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content headers = %#v", request.Header)
		}
		if request.Header.Get("User-Agent") != "transport-test" || request.Header.Get("X-Test") != "value" {
			t.Errorf("request headers = %#v", request.Header)
		}
		data, err := io.ReadAll(request.Body)
		if err != nil {
			t.Errorf("read body: %v", err)
		}
		if string(data) != "payload" {
			t.Errorf("body = %q", data)
		}

		writer.Header().Set("X-Result", "ok")
		writer.WriteHeader(http.StatusCreated)
		_, _ = writer.Write([]byte(`{"id":"1"}`))
	}))
	t.Cleanup(server.Close)

	transport := Transport{
		BaseURL:    server.URL + "/base",
		HTTPClient: server.Client(),
		UserAgent:  "transport-test",
		Header:     http.Header{"X-Test": {"value"}},
	}
	result, err := transport.Do(
		context.Background(),
		http.MethodPost,
		"/resource",
		url.Values{"cursor": {"next"}},
		strings.NewReader("payload"),
		"application/json",
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.StatusCode != http.StatusCreated || result.Header.Get("X-Result") != "ok" || string(result.Body) != `{"id":"1"}` {
		t.Fatalf("result = %#v", result)
	}
}

func TestNewHTTPClientTimeout(t *testing.T) {
	t.Parallel()

	if timeout := NewHTTPClient().Timeout; timeout != 30*time.Second {
		t.Fatalf("timeout = %v", timeout)
	}
}
