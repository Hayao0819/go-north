package clientconfig

import (
	"net/http"
	"testing"
)

func TestNew(t *testing.T) {
	t.Parallel()

	config := New("https://api.example", "example")
	if config.BaseURL != "https://api.example" || config.UserAgent != "example" {
		t.Fatalf("default config = %#v", config)
	}
	if config.HTTPClient == nil {
		t.Fatal("default HTTP client is nil")
	}
}

func TestSetters(t *testing.T) {
	t.Parallel()

	config := New("https://api.example", "example")
	client := &http.Client{}
	if err := config.SetBaseURL("https://example.com/api/"); err != nil {
		t.Fatal(err)
	}
	if err := config.SetHTTPClient(client); err != nil {
		t.Fatal(err)
	}
	if err := config.SetUserAgent("test"); err != nil {
		t.Fatal(err)
	}
	if config.BaseURL != "https://example.com/api" || config.HTTPClient != client || config.UserAgent != "test" {
		t.Fatalf("config = %#v", config)
	}

	if err := config.SetHTTPClient(nil); err == nil {
		t.Error("SetHTTPClient accepted nil")
	}
	if err := config.SetUserAgent("bad\nagent"); err == nil {
		t.Error("SetUserAgent accepted a newline")
	}
	if err := config.SetBaseURL("relative"); err == nil {
		t.Error("SetBaseURL accepted a relative URL")
	}
}
