package north

import (
	"context"
	"net/http"
	"net/url"
)

// TimelineOptions controls the home timeline.
type TimelineOptions struct {
	Ranked bool
	Cursor string
}

// HomeTimeline returns the caller's chronological or ranked home timeline.
func (c *Client) HomeTimeline(ctx context.Context, opts TimelineOptions) (PostPage, *Response, error) {
	query := url.Values{}
	if opts.Ranked {
		query.Set("ranked", "1")
	}
	setCursor(query, opts.Cursor)

	return doData[PostPage](ctx, c, http.MethodGet, "/2/timelines/home", query, nil, "")
}
