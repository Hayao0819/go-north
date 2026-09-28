package north

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

// Users fetches up to 100 accounts in the same order as handles. A leading @
// is accepted by the service.
func (c *Client) Users(ctx context.Context, handles ...string) ([]User, *Response, error) {
	if len(handles) == 0 || len(handles) > 100 {
		return nil, nil, errors.New("north: Users requires between 1 and 100 handles")
	}

	data, resp, err := doData[struct {
		Items []User `json:"items"`
	}](ctx, c, http.MethodGet, "/2/users", url.Values{"handles": {strings.Join(handles, ",")}}, nil, "")

	return data.Items, resp, err
}

// Me fetches the account that owns the API key.
func (c *Client) Me(ctx context.Context) (User, *Response, error) {
	return doData[User](ctx, c, http.MethodGet, "/2/users/me", nil, nil, "")
}

// User fetches a public account profile.
func (c *Client) User(ctx context.Context, handle string) (User, *Response, error) {
	return doData[User](ctx, c, http.MethodGet, userEndpoint(handle, ""), nil, nil, "")
}

// UserPosts returns posts by an account.
func (c *Client) UserPosts(ctx context.Context, handle, cursor string) (PostPage, *Response, error) {
	return c.userPostPage(ctx, handle, "tweets", cursor)
}

// Mentions returns public posts containing @handle. It is not the private
// notification feed.
func (c *Client) Mentions(ctx context.Context, handle, cursor string) (PostPage, *Response, error) {
	return c.userPostPage(ctx, handle, "mentions", cursor)
}

// LikedPosts returns public posts liked by an account.
func (c *Client) LikedPosts(ctx context.Context, handle, cursor string) (PostPage, *Response, error) {
	return c.userPostPage(ctx, handle, "liked_tweets", cursor)
}

func (c *Client) userPostPage(ctx context.Context, handle, resource, cursor string) (PostPage, *Response, error) {
	q := url.Values{}
	setCursor(q, cursor)

	return doData[PostPage](ctx, c, http.MethodGet, userEndpoint(handle, resource), q, nil, "")
}

// Followers returns one page of an account's followers.
func (c *Client) Followers(ctx context.Context, handle, cursor string) (UserPage, *Response, error) {
	return c.userPage(ctx, handle, "followers", cursor)
}

// Following returns one page of accounts followed by handle.
func (c *Client) Following(ctx context.Context, handle, cursor string) (UserPage, *Response, error) {
	return c.userPage(ctx, handle, "following", cursor)
}

func (c *Client) userPage(ctx context.Context, handle, resource, cursor string) (UserPage, *Response, error) {
	q := url.Values{}
	setCursor(q, cursor)

	return doData[UserPage](ctx, c, http.MethodGet, userEndpoint(handle, resource), q, nil, "")
}

// Follow follows an account. A protected account receives a follow request.
func (c *Client) Follow(ctx context.Context, handle string) (bool, *Response, error) {
	return c.setRelation(ctx, handle, "follow", true)
}

// Unfollow stops following an account.
func (c *Client) Unfollow(ctx context.Context, handle string) (bool, *Response, error) {
	return c.setRelation(ctx, handle, "follow", false)
}

// Block blocks an account and removes follows in both directions.
func (c *Client) Block(ctx context.Context, handle string) (bool, *Response, error) {
	return c.setRelation(ctx, handle, "block", true)
}

// Unblock removes an account block.
func (c *Client) Unblock(ctx context.Context, handle string) (bool, *Response, error) {
	return c.setRelation(ctx, handle, "block", false)
}

// Mute hides an account from timelines and notifications without unfollowing.
func (c *Client) Mute(ctx context.Context, handle string) (bool, *Response, error) {
	return c.setRelation(ctx, handle, "mute", true)
}

// Unmute removes an account mute.
func (c *Client) Unmute(ctx context.Context, handle string) (bool, *Response, error) {
	return c.setRelation(ctx, handle, "mute", false)
}

func (c *Client) setRelation(ctx context.Context, handle, relation string, on bool) (bool, *Response, error) {
	method := http.MethodDelete
	if on {
		method = http.MethodPost
	}

	data, resp, err := doData[struct {
		OK bool `json:"ok"`
	}](ctx, c, method, userEndpoint(handle, relation), nil, nil, "")

	return data.OK, resp, err
}

func userEndpoint(handle, resource string) string {
	handle = strings.TrimPrefix(handle, "@")
	path := "/2/users/" + url.PathEscape(handle)
	if resource != "" {
		path += "/" + resource
	}

	return path
}
