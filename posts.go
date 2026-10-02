package north

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ReplyPolicy is the audience allowed to reply to a post.
type ReplyPolicy string

const (
	ReplyEveryone  ReplyPolicy = "EVERYONE"
	ReplyFollowing ReplyPolicy = "FOLLOWING"
	ReplyMentioned ReplyPolicy = "MENTIONED"
)

// HiddenReason is why a post would normally be hidden.
type HiddenReason string

const (
	HiddenMuted   HiddenReason = "muted"
	HiddenBlocked HiddenReason = "blocked"
)

// Post is a north post. Quoted and RepostOf are expanded by at most one level
// by the service.
type Post struct {
	ID                string       `json:"id"`
	Text              string       `json:"text"`
	CreatedAt         time.Time    `json:"createdAt"`
	EditedAt          *time.Time   `json:"editedAt"`
	EditCount         int          `json:"editCount"`
	Author            User         `json:"author"`
	ConversationID    string       `json:"conversationId"`
	ReplyPolicy       ReplyPolicy  `json:"replyPolicy"`
	Source            string       `json:"source"`
	InReplyToID       *string      `json:"inReplyToId"`
	InReplyToHandle   *string      `json:"inReplyToHandle"`
	Quoted            *Post        `json:"quoted"`
	RepostOf          *Post        `json:"retweetOf"`
	LikeCount         int          `json:"likeCount"`
	RepostCount       int          `json:"retweetCount"`
	ReplyCount        int          `json:"replyCount"`
	QuoteCount        int          `json:"quoteCount"`
	Media             []Media      `json:"media"`
	Poll              *Poll        `json:"poll"`
	Liked             bool         `json:"liked"`
	Reposted          bool         `json:"retweeted"`
	Bookmarked        bool         `json:"bookmarked"`
	Deleted           bool         `json:"deleted"`
	Unavailable       bool         `json:"unavailable"`
	QuotedUnavailable bool         `json:"quotedUnavailable"`
	HiddenReason      HiddenReason `json:"hiddenReason"`
	EditEligible      bool         `json:"editEligible"`
}

// DisplayPost returns the original post for a repost, and the receiver for a
// normal post. It is useful for clients that render repost attribution around
// the original content.
func (p *Post) DisplayPost() *Post {
	if p != nil && p.RepostOf != nil {
		return p.RepostOf
	}

	return p
}

// Poll is a poll attached to a post.
type Poll struct {
	ID             string       `json:"id"`
	EndsAt         time.Time    `json:"endsAt"`
	Ended          bool         `json:"ended"`
	TotalVotes     int          `json:"totalVotes"`
	ViewerOptionID *string      `json:"viewerOptionId"`
	Options        []PollOption `json:"options"`
}

// PollOption is one answer in a poll.
type PollOption struct {
	ID        string  `json:"id"`
	Label     string  `json:"label"`
	Position  int     `json:"position"`
	VoteCount int     `json:"voteCount"`
	Percent   float64 `json:"percent"`
}

// PostPage is one cursor-paginated page of posts.
type PostPage struct {
	Items      []Post  `json:"items"`
	NextCursor *string `json:"nextCursor"`
}

// CreatePostRequest is the request body for a new post. Text and Media are
// individually optional, but the API requires at least one of them.
type CreatePostRequest struct {
	Text        string           `json:"text,omitempty"`
	Media       *CreatePostMedia `json:"media,omitempty"`
	Reply       *CreatePostReply `json:"reply,omitempty"`
	QuotePostID string           `json:"quote_tweet_id,omitempty"`
	Poll        *CreatePoll      `json:"poll,omitempty"`
}

// CreatePoll describes a poll attached to a new post.
type CreatePoll struct {
	Options         []string `json:"options"`
	DurationMinutes int      `json:"durationMinutes"`
}

// CreatePostMedia contains IDs returned by a media upload. The API accepts at
// most four IDs.
type CreatePostMedia struct {
	MediaIDs []string `json:"media_ids"`
}

// CreatePostReply identifies the post being replied to.
type CreatePostReply struct {
	InReplyToPostID string `json:"in_reply_to_tweet_id"`
}

// CreatedPost is the compact response returned after creating a post.
type CreatedPost struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// LikeState is the state after liking or unliking a post.
type LikeState struct {
	Liked     bool `json:"liked"`
	LikeCount int  `json:"likeCount"`
}

// RepostState is the state after reposting or undoing a repost.
type RepostState struct {
	Reposted    bool `json:"retweeted"`
	RepostCount int  `json:"retweetCount"`
}

// EditPostRequest is the editable part of a post. A non-nil empty MediaIDs
// removes every attachment.
type EditPostRequest struct {
	Text     *string   `json:"text,omitempty"`
	MediaIDs *[]string `json:"mediaIds,omitempty"`
}

// ThreadItem is one post in a new thread.
type ThreadItem struct {
	Text        string      `json:"text,omitempty"`
	MediaIDs    []string    `json:"mediaIds,omitempty"`
	ReplyPolicy ReplyPolicy `json:"replyPolicy,omitempty"`
	Poll        *CreatePoll `json:"poll,omitempty"`
}

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

// Post fetches one post.
func (c *Client) Post(ctx context.Context, id string) (Post, *Response, error) {
	return doData[Post](ctx, c, http.MethodGet, "/2/tweets/"+url.PathEscape(id), nil, nil, "")
}

// EditPost edits one of the caller's posts.
func (c *Client) EditPost(ctx context.Context, id string, req EditPostRequest) (Post, *Response, error) {
	return c.editPost(ctx, id, req, "")
}

// EditPostIfMatch edits a post only when etag still matches the current
// version.
func (c *Client) EditPostIfMatch(ctx context.Context, id string, req EditPostRequest, etag string) (Post, *Response, error) {
	return c.editPost(ctx, id, req, etag)
}

func (c *Client) editPost(ctx context.Context, id string, req EditPostRequest, etag string) (Post, *Response, error) {
	body, contentType, err := jsonRequest(req)
	if err != nil {
		return Post{}, nil, err
	}
	header := make(http.Header)
	if etag != "" {
		header.Set("If-Match", etag)
	}

	return doDataWithHeader[Post](ctx, c, http.MethodPut, "/2/tweets/"+url.PathEscape(id)+"/edit", nil, body, contentType, header)
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

// VotePoll votes for one option in a post's poll.
func (c *Client) VotePoll(ctx context.Context, id, optionID string) (Post, *Response, error) {
	body, contentType, err := jsonRequest(struct {
		OptionID string `json:"optionId"`
	}{OptionID: optionID})
	if err != nil {
		return Post{}, nil, err
	}

	return doData[Post](ctx, c, http.MethodPost, "/2/tweets/"+url.PathEscape(id)+"/poll/vote", nil, body, contentType)
}

// CreateThread publishes up to 25 connected posts.
func (c *Client) CreateThread(ctx context.Context, items []ThreadItem) ([]Post, *Response, error) {
	if len(items) == 0 || len(items) > 25 {
		return nil, nil, errors.New("north: CreateThread requires between 1 and 25 items")
	}
	body, contentType, err := jsonRequest(struct {
		Items []ThreadItem `json:"items"`
	}{Items: items})
	if err != nil {
		return nil, nil, err
	}
	data, response, err := doData[struct {
		Items []Post `json:"items"`
	}](ctx, c, http.MethodPost, "/2/tweets/thread", nil, body, contentType)

	return data.Items, response, err
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
