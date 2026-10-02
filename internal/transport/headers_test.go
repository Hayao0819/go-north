package transport

import (
	"net/http"
	"testing"
	"time"
)

func TestRetryAfter(t *testing.T) {
	t.Parallel()

	for value, want := range map[string]time.Duration{
		"":   0,
		"-1": 0,
		"x":  0,
		"3":  3 * time.Second,
	} {
		header := make(http.Header)
		header.Set("Retry-After", value)
		if got := RetryAfter(header); got != want {
			t.Errorf("RetryAfter(%q) = %v, want %v", value, got, want)
		}
	}
}
