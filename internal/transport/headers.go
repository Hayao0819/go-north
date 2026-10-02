package transport

import (
	"net/http"
	"strconv"
	"time"
)

// RetryAfter parses a Retry-After header containing a delay in seconds.
func RetryAfter(header http.Header) time.Duration {
	seconds, err := strconv.ParseInt(header.Get("Retry-After"), 10, 64)
	if err != nil || seconds < 0 || seconds > int64((time.Duration(1<<63-1))/time.Second) {
		return 0
	}

	return time.Duration(seconds) * time.Second
}
