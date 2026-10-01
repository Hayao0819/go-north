package unofficial

import (
	"context"
	"net/http"
	"net/url"
)

// PostConversationPage is one page of the conversation around a post.
type PostConversationPage struct {
	Ancestors    []Post  `json:"ancestors"`
	Tweet        Post    `json:"tweet"`
	Replies      []Post  `json:"replies"`
	NextCursor   *string `json:"nextCursor"`
	ReaderRootID *string `json:"readerRootId"`
}

// PostReplyPage is the continuation of a same-author reply chain.
type PostReplyPage struct {
	Items     []Post `json:"items"`
	Truncated bool   `json:"truncated"`
}

// PostConversation returns the post, its ancestors, and a page of replies.
func (c *Client) PostConversation(ctx context.Context, id, cursor string) (PostConversationPage, *Response, error) {
	query := make(url.Values)
	setCursor(query, cursor)

	return doJSON[PostConversationPage](ctx, c, http.MethodGet, postEndpoint(id)+"/conversation", query, nil)
}

// PostReplies returns the same-author reply chain which follows a post.
func (c *Client) PostReplies(ctx context.Context, id string) (PostReplyPage, *Response, error) {
	return doJSON[PostReplyPage](ctx, c, http.MethodGet, postEndpoint(id)+"/replies", nil, nil)
}

func postEndpoint(id string) string {
	return "/api/tweets/" + url.PathEscape(id)
}
