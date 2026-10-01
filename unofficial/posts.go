package unofficial

import (
	"context"
	"net/http"
	"net/url"

	"github.com/Hayao0819/go-north"
)

// HomeTimeline returns the current account's chronological or ranked timeline.
func (c *Client) HomeTimeline(ctx context.Context, options north.TimelineOptions) (PostPage, *Response, error) {
	ranked := "0"
	if options.Ranked {
		ranked = "1"
	}
	query := url.Values{"ranked": {ranked}}
	setCursor(query, options.Cursor)

	return doJSON[PostPage](ctx, c, http.MethodGet, "/api/timeline/home", query, nil)
}

// SearchPosts searches posts in the same order as the north web client.
func (c *Client) SearchPosts(ctx context.Context, value string, options north.SearchOptions) (PostPage, *Response, error) {
	tab := options.Tab
	if tab == "" {
		tab = north.SearchLatest
	}
	query := url.Values{
		"q":   {value},
		"tab": {string(tab)},
	}
	setCursor(query, options.Cursor)

	return doJSON[PostPage](ctx, c, http.MethodGet, "/api/search", query, nil)
}

// Post fetches one post including web-only state such as edit eligibility.
func (c *Client) Post(ctx context.Context, id string) (Post, *Response, error) {
	return doJSON[Post](ctx, c, http.MethodGet, "/api/tweets/"+url.PathEscape(id), nil, nil)
}

// CreatePost publishes a post through the web API.
func (c *Client) CreatePost(ctx context.Context, request CreatePostRequest) (north.CreatedPost, *Response, error) {
	return doJSON[north.CreatedPost](ctx, c, http.MethodPost, "/api/tweets", nil, request)
}

// DeletePost deletes one of the current account's posts.
func (c *Client) DeletePost(ctx context.Context, id string) (bool, *Response, error) {
	response, err := doEmpty(ctx, c, http.MethodDelete, "/api/tweets/"+url.PathEscape(id), nil, nil)

	return err == nil, response, err
}

// Like likes a post.
func (c *Client) Like(ctx context.Context, id string) (north.LikeState, *Response, error) {
	return c.setLike(ctx, id, true)
}

// Unlike removes the current account's like from a post.
func (c *Client) Unlike(ctx context.Context, id string) (north.LikeState, *Response, error) {
	return c.setLike(ctx, id, false)
}

func (c *Client) setLike(ctx context.Context, id string, liked bool) (north.LikeState, *Response, error) {
	method := http.MethodDelete
	if liked {
		method = http.MethodPost
	}

	return doJSON[north.LikeState](ctx, c, method, "/api/tweets/"+url.PathEscape(id)+"/like", nil, nil)
}

// Repost reposts a post.
func (c *Client) Repost(ctx context.Context, id string) (north.RepostState, *Response, error) {
	return c.setRepost(ctx, id, true)
}

// UndoRepost removes the current account's repost.
func (c *Client) UndoRepost(ctx context.Context, id string) (north.RepostState, *Response, error) {
	return c.setRepost(ctx, id, false)
}

func (c *Client) setRepost(ctx context.Context, id string, reposted bool) (north.RepostState, *Response, error) {
	method := http.MethodDelete
	if reposted {
		method = http.MethodPost
	}

	return doJSON[north.RepostState](ctx, c, method, "/api/tweets/"+url.PathEscape(id)+"/retweet", nil, nil)
}

// EditPost replaces the editable text and media of a post.
func (c *Client) EditPost(ctx context.Context, id string, request EditPostRequest) (*Response, error) {
	if request.MediaIDs == nil {
		request.MediaIDs = []string{}
	}

	return doEmpty(ctx, c, http.MethodPut, "/api/tweets/"+url.PathEscape(id)+"/edit", nil, request)
}

// PostEditHistory returns every retained version of a post.
func (c *Client) PostEditHistory(ctx context.Context, id string) (PostEditHistory, *Response, error) {
	return doJSON[PostEditHistory](ctx, c, http.MethodGet, "/api/tweets/"+url.PathEscape(id)+"/edit-history", nil, nil)
}

// VotePoll votes for an option in a post's poll.
func (c *Client) VotePoll(ctx context.Context, id, optionID string) (Poll, *Response, error) {
	data, response, err := doJSON[struct {
		Poll Poll `json:"poll"`
	}](ctx, c, http.MethodPost, "/api/tweets/"+url.PathEscape(id)+"/poll/vote", nil, struct {
		OptionID string `json:"optionId"`
	}{OptionID: optionID})

	return data.Poll, response, err
}

func setCursor(query url.Values, cursor string) {
	if cursor != "" {
		query.Set("cursor", cursor)
	}
}
