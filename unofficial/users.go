package unofficial

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/Hayao0819/go-north"
)

// User fetches a public account profile.
func (c *Client) User(ctx context.Context, handle string) (User, *Response, error) {
	return doJSON[User](ctx, c, http.MethodGet, userEndpoint(handle, ""), nil, nil)
}

// UserPosts returns posts from an account's profile.
func (c *Client) UserPosts(ctx context.Context, handle, cursor string) (PostPage, *Response, error) {
	query := url.Values{"tab": {"tweets"}}
	setCursor(query, cursor)

	return doJSON[PostPage](ctx, c, http.MethodGet, userEndpoint(handle, "tweets"), query, nil)
}

// Mentions returns the latest posts that mention an account.
func (c *Client) Mentions(ctx context.Context, handle, cursor string) (PostPage, *Response, error) {
	handle = strings.TrimPrefix(strings.TrimSpace(handle), "@")

	return c.SearchPosts(ctx, "@"+handle, north.SearchOptions{Tab: north.SearchLatest, Cursor: cursor})
}

// UpdateProfile changes fields on the current account's public profile.
func (c *Client) UpdateProfile(ctx context.Context, request UpdateProfileRequest) (*Response, error) {
	return doEmpty(ctx, c, http.MethodPatch, "/api/users/me", nil, request)
}

func userEndpoint(handle, resource string) string {
	handle = strings.TrimPrefix(strings.TrimSpace(handle), "@")
	endpoint := "/api/users/" + url.PathEscape(handle)
	if resource != "" {
		endpoint += "/" + resource
	}

	return endpoint
}
