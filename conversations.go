package north

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// Conversation contains a post, its ancestors, and the first page of replies.
type Conversation struct {
	Ancestors  []Post  `json:"ancestors"`
	Post       Post    `json:"tweet"`
	Replies    []Post  `json:"replies"`
	NextCursor *string `json:"nextCursor"`
}

// ReplyPage is one page of direct replies.
type ReplyPage struct {
	Items      []Post  `json:"items"`
	NextCursor *string `json:"nextCursor"`
	Truncated  bool    `json:"truncated"`
}

// EditHistory contains every retained version of a post.
type EditHistory struct {
	Post     Post          `json:"tweet"`
	Versions []PostVersion `json:"versions"`
}

// PostVersion is one entry in a post's edit history.
type PostVersion struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	Media     []Media   `json:"media"`
	CreatedAt time.Time `json:"createdAt"`
	Current   bool      `json:"current"`
}

// Conversation returns a post with up to 50 ancestors and ten replies.
func (c *Client) Conversation(ctx context.Context, id, cursor string) (Conversation, *Response, error) {
	query := url.Values{}
	setCursor(query, cursor)

	return doData[Conversation](ctx, c, http.MethodGet, "/2/tweets/"+url.PathEscape(id)+"/conversation", query, nil, "")
}

// Replies returns direct replies to a post.
func (c *Client) Replies(ctx context.Context, id, cursor string) (ReplyPage, *Response, error) {
	query := url.Values{}
	setCursor(query, cursor)

	return doData[ReplyPage](ctx, c, http.MethodGet, "/2/tweets/"+url.PathEscape(id)+"/replies", query, nil, "")
}

// PostEditHistory returns the edit history for a post.
func (c *Client) PostEditHistory(ctx context.Context, id string) (EditHistory, *Response, error) {
	return doData[EditHistory](ctx, c, http.MethodGet, "/2/tweets/"+url.PathEscape(id)+"/edit-history", nil, nil, "")
}

// PostLikes returns accounts that liked a post.
func (c *Client) PostLikes(ctx context.Context, id, cursor string) (UserPage, *Response, error) {
	return c.postUsers(ctx, id, "likes", cursor)
}

// PostReposts returns accounts that reposted a post.
func (c *Client) PostReposts(ctx context.Context, id, cursor string) (UserPage, *Response, error) {
	return c.postUsers(ctx, id, "retweets", cursor)
}

func (c *Client) postUsers(ctx context.Context, id, resource, cursor string) (UserPage, *Response, error) {
	query := url.Values{}
	setCursor(query, cursor)

	return doData[UserPage](ctx, c, http.MethodGet, "/2/tweets/"+url.PathEscape(id)+"/"+resource, query, nil, "")
}
