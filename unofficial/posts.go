package unofficial

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/Hayao0819/go-north"
)

// Post adds web-only fields to the public API's post representation.
type Post struct {
	north.Post
	Author             User         `json:"author"`
	Media              []Media      `json:"media"`
	Poll               *Poll        `json:"poll"`
	Quoted             *Post        `json:"quoted"`
	RepostOf           *Post        `json:"retweetOf"`
	EditEligible       bool         `json:"editEligible"`
	Pinned             bool         `json:"pinned"`
	LinkPreview        *LinkPreview `json:"linkPreview"`
	ThreadContinuation []Post       `json:"threadContinuation"`
}

// DisplayPost returns the original post for a repost.
func (p *Post) DisplayPost() *Post {
	if p != nil && p.RepostOf != nil {
		return p.RepostOf
	}

	return p
}

// PublicPost returns the fields shared with the public API representation.
func (p Post) PublicPost() north.Post {
	result := p.Post
	result.Author = p.Author.User
	result.EditEligible = p.EditEligible
	if p.Media != nil {
		result.Media = make([]north.Media, len(p.Media))
		for index := range p.Media {
			result.Media[index] = p.Media[index].Media
		}
	}
	if p.Poll != nil {
		poll := p.Poll.publicPoll()
		result.Poll = &poll
	}
	if p.Quoted != nil {
		quoted := p.Quoted.PublicPost()
		result.Quoted = &quoted
	}
	if p.RepostOf != nil {
		repost := p.RepostOf.PublicPost()
		result.RepostOf = &repost
	}

	return result
}

// LinkPreviewCard controls how a link preview is displayed.
type LinkPreviewCard string

const (
	// LinkPreviewSummary displays a compact preview with a thumbnail.
	LinkPreviewSummary LinkPreviewCard = "summary"
	// LinkPreviewSummaryLargeImage displays a preview with a large image.
	LinkPreviewSummaryLargeImage LinkPreviewCard = "summary_large_image"
)

// LinkPreview contains metadata fetched for a URL in a post.
type LinkPreview struct {
	URL         string          `json:"url"`
	Title       *string         `json:"title"`
	Description *string         `json:"description"`
	ImageURL    *string         `json:"imageUrl"`
	Domain      string          `json:"domain"`
	Card        LinkPreviewCard `json:"card"`
}

// Media adds fields returned by the web API to the public media type.
type Media struct {
	north.Media
	ErrorMessage *string `json:"errorMessage"`
	Position     int     `json:"position"`
}

// Poll is a poll attached to a web API post.
type Poll struct {
	ID             string       `json:"id"`
	EndsAt         time.Time    `json:"endsAt"`
	Ended          bool         `json:"ended"`
	TotalVotes     int          `json:"totalVotes"`
	ViewerOptionID *string      `json:"viewerOptionId"`
	Options        []PollOption `json:"options"`
}

func (p Poll) publicPoll() north.Poll {
	result := north.Poll{
		ID:             p.ID,
		EndsAt:         p.EndsAt,
		Ended:          p.Ended,
		TotalVotes:     p.TotalVotes,
		ViewerOptionID: p.ViewerOptionID,
		Options:        make([]north.PollOption, len(p.Options)),
	}
	for index, option := range p.Options {
		result.Options[index] = north.PollOption{
			ID:        option.ID,
			Label:     option.Label,
			Position:  option.Position,
			VoteCount: option.VoteCount,
			Percent:   float64(option.Percent),
		}
	}

	return result
}

// PollOption is one choice in a web API poll.
type PollOption struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Position  int    `json:"position"`
	VoteCount int    `json:"voteCount"`
	Percent   int    `json:"percent"`
}

// PostPage is one cursor-paginated page of posts.
type PostPage struct {
	Items      []Post     `json:"items"`
	NextCursor *string    `json:"nextCursor"`
	NewestAt   *time.Time `json:"newestAt"`
}

// PublicPostPage returns the fields shared with a public API post page.
func (p PostPage) PublicPostPage() north.PostPage {
	items := make([]north.Post, len(p.Items))
	for index := range p.Items {
		items[index] = p.Items[index].PublicPost()
	}

	return north.PostPage{Items: items, NextCursor: p.NextCursor}
}

// CreatePostRequest is the request used by the web client when publishing.
type CreatePostRequest struct {
	Text        string             `json:"text,omitempty"`
	ReplyPolicy north.ReplyPolicy  `json:"replyPolicy,omitempty"`
	MediaIDs    []string           `json:"mediaIds,omitempty"`
	InReplyToID string             `json:"inReplyToId,omitempty"`
	QuotedID    string             `json:"quotedId,omitempty"`
	Poll        *CreatePollRequest `json:"poll,omitempty"`
}

// CreatePollRequest contains the choices and duration of a new poll.
type CreatePollRequest struct {
	Options         []string `json:"options"`
	DurationMinutes int      `json:"durationMinutes"`
}

// EditPostRequest is the editable portion of a post.
type EditPostRequest struct {
	Text     string   `json:"text"`
	MediaIDs []string `json:"mediaIds"`
}

// PostEditHistory contains the current post and its retained versions.
type PostEditHistory struct {
	Post     Post              `json:"tweet"`
	Versions []PostEditVersion `json:"versions"`
}

// PostEditVersion is one entry in a post's edit history.
type PostEditVersion struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	Media     []Media   `json:"media"`
	CreatedAt time.Time `json:"createdAt"`
	Current   bool      `json:"current"`
}

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
	return doJSON[Post](ctx, c, http.MethodGet, postEndpoint(id), nil, nil)
}

// CreatePost publishes a post through the web API.
func (c *Client) CreatePost(ctx context.Context, request CreatePostRequest) (north.CreatedPost, *Response, error) {
	return doJSON[north.CreatedPost](ctx, c, http.MethodPost, "/api/tweets", nil, request)
}

// DeletePost deletes one of the current account's posts.
func (c *Client) DeletePost(ctx context.Context, id string) (bool, *Response, error) {
	response, err := doEmpty(ctx, c, http.MethodDelete, postEndpoint(id), nil, nil)

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

	return doJSON[north.LikeState](ctx, c, method, postEndpoint(id)+"/like", nil, nil)
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

	return doJSON[north.RepostState](ctx, c, method, postEndpoint(id)+"/retweet", nil, nil)
}

// EditPost replaces the editable text and media of a post.
func (c *Client) EditPost(ctx context.Context, id string, request EditPostRequest) (*Response, error) {
	if request.MediaIDs == nil {
		request.MediaIDs = []string{}
	}

	return doEmpty(ctx, c, http.MethodPut, postEndpoint(id)+"/edit", nil, request)
}

// PostEditHistory returns every retained version of a post.
func (c *Client) PostEditHistory(ctx context.Context, id string) (PostEditHistory, *Response, error) {
	return doJSON[PostEditHistory](ctx, c, http.MethodGet, postEndpoint(id)+"/edit-history", nil, nil)
}

// VotePoll votes for an option in a post's poll.
func (c *Client) VotePoll(ctx context.Context, id, optionID string) (Poll, *Response, error) {
	data, response, err := doJSON[struct {
		Poll Poll `json:"poll"`
	}](ctx, c, http.MethodPost, postEndpoint(id)+"/poll/vote", nil, struct {
		OptionID string `json:"optionId"`
	}{OptionID: optionID})

	return data.Poll, response, err
}

func setCursor(query url.Values, cursor string) {
	if cursor != "" {
		query.Set("cursor", cursor)
	}
}
