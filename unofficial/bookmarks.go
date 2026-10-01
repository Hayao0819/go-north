package unofficial

import (
	"context"
	"net/http"
	"net/url"
)

// Bookmarks returns one page of the current account's bookmarks.
func (c *Client) Bookmarks(ctx context.Context, cursor string) (PostPage, *Response, error) {
	query := url.Values{}
	setCursor(query, cursor)

	return doJSON[PostPage](ctx, c, http.MethodGet, "/api/bookmarks", query, nil)
}

// Bookmark adds a post to the current account's bookmarks.
func (c *Client) Bookmark(ctx context.Context, id string) (*Response, error) {
	return doEmpty(ctx, c, http.MethodPost, "/api/tweets/"+url.PathEscape(id)+"/bookmark", nil, nil)
}

// Unbookmark removes a post from the current account's bookmarks.
func (c *Client) Unbookmark(ctx context.Context, id string) (*Response, error) {
	return doEmpty(ctx, c, http.MethodDelete, "/api/tweets/"+url.PathEscape(id)+"/bookmark", nil, nil)
}
