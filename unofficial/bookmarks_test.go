package unofficial

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

func TestBookmarkEndpoints(t *testing.T) {
	t.Parallel()

	testEndpoints(t, []endpointTest{
		{"list", http.MethodGet, "/api/bookmarks", url.Values{"cursor": {"next"}}, nil, func(ctx context.Context, client *Client) error {
			_, _, err := client.Bookmarks(ctx, "next")
			return err
		}},
		{"add", http.MethodPost, "/api/tweets/1/bookmark", nil, nil, func(ctx context.Context, client *Client) error {
			_, err := client.Bookmark(ctx, "1")
			return err
		}},
		{"remove", http.MethodDelete, "/api/tweets/1/bookmark", nil, nil, func(ctx context.Context, client *Client) error {
			_, err := client.Unbookmark(ctx, "1")
			return err
		}},
	})
}
