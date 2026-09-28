package north

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
)

// Posts fetches up to 100 posts in the same order as ids. Inaccessible and
// missing posts are omitted by the service.
func (c *Client) Posts(ctx context.Context, ids ...string) ([]Post, *Response, error) {
	if len(ids) == 0 || len(ids) > 100 {
		return nil, nil, errors.New("north: Posts requires between 1 and 100 IDs")
	}

	data, resp, err := doData[struct {
		Items []Post `json:"items"`
	}](ctx, c, http.MethodGet, "/2/tweets", url.Values{"ids": {strings.Join(ids, ",")}}, nil, "")

	return data.Items, resp, err
}

// CreatePost publishes a post.
func (c *Client) CreatePost(ctx context.Context, req CreatePostRequest) (CreatedPost, *Response, error) {
	body, contentType, err := jsonRequest(req)
	if err != nil {
		return CreatedPost{}, nil, err
	}

	return doData[CreatedPost](ctx, c, http.MethodPost, "/2/tweets", nil, body, contentType)
}

// SearchPosts searches public posts. North accepts the same query syntax as
// its web search, including from:handle and hashtags.
func (c *Client) SearchPosts(ctx context.Context, query string, opts SearchOptions) (PostPage, *Response, error) {
	q := url.Values{"q": {query}}
	if opts.Tab != "" {
		q.Set("tab", string(opts.Tab))
	}
	setCursor(q, opts.Cursor)

	return doData[PostPage](ctx, c, http.MethodGet, "/2/tweets/search", q, nil, "")
}

// CountPosts returns the number of posts matching query.
func (c *Client) CountPosts(ctx context.Context, query string) (int, *Response, error) {
	data, resp, err := doData[struct {
		Count int `json:"count"`
	}](ctx, c, http.MethodGet, "/2/tweets/counts", url.Values{"q": {query}}, nil, "")

	return data.Count, resp, err
}

// Post fetches one post.
func (c *Client) Post(ctx context.Context, id string) (Post, *Response, error) {
	return doData[Post](ctx, c, http.MethodGet, "/2/tweets/"+url.PathEscape(id), nil, nil, "")
}

// DeletePost deletes one of the caller's posts.
func (c *Client) DeletePost(ctx context.Context, id string) (bool, *Response, error) {
	data, resp, err := doData[struct {
		Deleted bool `json:"deleted"`
	}](ctx, c, http.MethodDelete, "/2/tweets/"+url.PathEscape(id), nil, nil, "")

	return data.Deleted, resp, err
}

// Quotes returns posts quoting id.
func (c *Client) Quotes(ctx context.Context, id, cursor string) (PostPage, *Response, error) {
	q := url.Values{}
	setCursor(q, cursor)

	return doData[PostPage](ctx, c, http.MethodGet, "/2/tweets/"+url.PathEscape(id)+"/quotes", q, nil, "")
}

// HomeTimeline returns the caller's chronological or ranked home timeline.
func (c *Client) HomeTimeline(ctx context.Context, opts TimelineOptions) (PostPage, *Response, error) {
	q := url.Values{}
	if opts.Ranked {
		q.Set("ranked", "1")
	}
	setCursor(q, opts.Cursor)

	return doData[PostPage](ctx, c, http.MethodGet, "/2/timelines/home", q, nil, "")
}

// Like likes a post. Repeating the operation is idempotent.
func (c *Client) Like(ctx context.Context, id string) (LikeState, *Response, error) {
	return c.setLike(ctx, id, true)
}

// Unlike removes the caller's like from a post.
func (c *Client) Unlike(ctx context.Context, id string) (LikeState, *Response, error) {
	return c.setLike(ctx, id, false)
}

func (c *Client) setLike(ctx context.Context, id string, on bool) (LikeState, *Response, error) {
	method := http.MethodDelete
	if on {
		method = http.MethodPost
	}

	return doData[LikeState](ctx, c, method, "/2/tweets/"+url.PathEscape(id)+"/like", nil, nil, "")
}

// Repost reposts a post.
func (c *Client) Repost(ctx context.Context, id string) (RepostState, *Response, error) {
	return c.setRepost(ctx, id, true)
}

// UndoRepost removes the caller's repost.
func (c *Client) UndoRepost(ctx context.Context, id string) (RepostState, *Response, error) {
	return c.setRepost(ctx, id, false)
}

func (c *Client) setRepost(ctx context.Context, id string, on bool) (RepostState, *Response, error) {
	method := http.MethodDelete
	if on {
		method = http.MethodPost
	}

	return doData[RepostState](ctx, c, method, "/2/tweets/"+url.PathEscape(id)+"/retweet", nil, nil, "")
}

func setCursor(q url.Values, cursor string) {
	if cursor != "" {
		q.Set("cursor", cursor)
	}
}
