package north

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"
)

// DraftItemInput is one post in a draft, scheduled post, or undoable post.
type DraftItemInput struct {
	Text        string      `json:"text,omitempty"`
	MediaIDs    []string    `json:"mediaIds,omitempty"`
	ReplyPolicy ReplyPolicy `json:"replyPolicy,omitempty"`
}

// DraftItem is one expanded item in a saved draft.
type DraftItem struct {
	Text        string      `json:"text"`
	ReplyPolicy ReplyPolicy `json:"replyPolicy"`
	Media       []Media     `json:"media"`
}

// Draft is a saved post or thread.
type Draft struct {
	ID            string      `json:"id"`
	Items         []DraftItem `json:"items"`
	QuotedPost    *Post       `json:"quotedTweet"`
	InReplyToPost *Post       `json:"inReplyToTweet"`
	CreatedAt     time.Time   `json:"createdAt"`
	UpdatedAt     time.Time   `json:"updatedAt"`
}

// DraftRequest is used to create or replace a draft.
type DraftRequest struct {
	Items       []DraftItemInput `json:"items"`
	QuotedID    *NullableString  `json:"quotedId,omitempty"`
	InReplyToID *NullableString  `json:"inReplyToId,omitempty"`
}

// ScheduledPost is a post saved for later publication.
type ScheduledPost struct {
	Draft
	ScheduledAt   time.Time `json:"scheduledAt"`
	ScheduleError *string   `json:"scheduleError"`
}

// SchedulePostRequest creates or replaces a scheduled post.
type SchedulePostRequest struct {
	Item        DraftItemInput  `json:"item"`
	ScheduledAt time.Time       `json:"scheduledAt"`
	QuotedID    *NullableString `json:"quotedId,omitempty"`
	InReplyToID *NullableString `json:"inReplyToId,omitempty"`
	FromDraftID *NullableString `json:"fromDraftId,omitempty"`
}

// PendingPost can be undone until UndoUntil.
type PendingPost struct {
	Draft
	UndoUntil     time.Time `json:"undoUntil"`
	ScheduleError *string   `json:"scheduleError"`
}

// CreatePendingPostRequest creates an undoable post.
type CreatePendingPostRequest struct {
	Item        DraftItemInput  `json:"item"`
	InReplyToID *NullableString `json:"inReplyToId,omitempty"`
	FromDraftID *NullableString `json:"fromDraftId,omitempty"`
}

// PendingPostState reports whether an undoable post was sent immediately or
// remains queued.
type PendingPostState struct {
	Sent    bool `json:"sent"`
	Pending bool `json:"pending"`
}

// Drafts returns the caller's saved drafts.
func (c *Client) Drafts(ctx context.Context, cursor string) ([]Draft, *Response, error) {
	return doItemPage[Draft](ctx, c, "/2/drafts", cursor)
}

// CreateDraft saves a draft.
func (c *Client) CreateDraft(ctx context.Context, req DraftRequest) (Draft, *Response, error) {
	return c.writeDraft(ctx, http.MethodPost, "/2/drafts", req)
}

// UpdateDraft replaces a saved draft.
func (c *Client) UpdateDraft(ctx context.Context, id string, req DraftRequest) (Draft, *Response, error) {
	return c.writeDraft(ctx, http.MethodPatch, "/2/drafts/"+url.PathEscape(id), req)
}

func (c *Client) writeDraft(ctx context.Context, method, endpoint string, req DraftRequest) (Draft, *Response, error) {
	body, contentType, err := jsonRequest(req)
	if err != nil {
		return Draft{}, nil, err
	}

	return doData[Draft](ctx, c, method, endpoint, nil, body, contentType)
}

// DeleteDrafts deletes up to 200 drafts.
func (c *Client) DeleteDrafts(ctx context.Context, ids ...string) (int, *Response, error) {
	return deleteSavedPosts(ctx, c, "/2/drafts", ids)
}

// ScheduledPosts returns the caller's scheduled posts.
func (c *Client) ScheduledPosts(ctx context.Context, cursor string) ([]ScheduledPost, *Response, error) {
	return doItemPage[ScheduledPost](ctx, c, "/2/scheduled-posts", cursor)
}

// SchedulePost creates a scheduled post.
func (c *Client) SchedulePost(ctx context.Context, req SchedulePostRequest) (ScheduledPost, *Response, error) {
	return c.writeScheduledPost(ctx, http.MethodPost, "/2/scheduled-posts", req)
}

// UpdateScheduledPost replaces a scheduled post.
func (c *Client) UpdateScheduledPost(ctx context.Context, id string, req SchedulePostRequest) (ScheduledPost, *Response, error) {
	return c.writeScheduledPost(ctx, http.MethodPatch, "/2/scheduled-posts/"+url.PathEscape(id), req)
}

func (c *Client) writeScheduledPost(ctx context.Context, method, endpoint string, req SchedulePostRequest) (ScheduledPost, *Response, error) {
	body, contentType, err := jsonRequest(req)
	if err != nil {
		return ScheduledPost{}, nil, err
	}

	return doData[ScheduledPost](ctx, c, method, endpoint, nil, body, contentType)
}

// DeleteScheduledPosts deletes up to 200 scheduled posts.
func (c *Client) DeleteScheduledPosts(ctx context.Context, ids ...string) (int, *Response, error) {
	return deleteSavedPosts(ctx, c, "/2/scheduled-posts", ids)
}

// PendingPosts returns the caller's undoable posts.
func (c *Client) PendingPosts(ctx context.Context, cursor string) ([]PendingPost, *Response, error) {
	return doItemPage[PendingPost](ctx, c, "/2/pending-posts", cursor)
}

// CreatePendingPost creates an undoable post.
func (c *Client) CreatePendingPost(ctx context.Context, req CreatePendingPostRequest) (PendingPost, *Response, error) {
	body, contentType, err := jsonRequest(req)
	if err != nil {
		return PendingPost{}, nil, err
	}

	return doData[PendingPost](ctx, c, http.MethodPost, "/2/pending-posts", nil, body, contentType)
}

// SendPendingPost sends an undoable post immediately.
func (c *Client) SendPendingPost(ctx context.Context, id string) (PendingPostState, *Response, error) {
	return doData[PendingPostState](ctx, c, http.MethodPost, pendingPostEndpoint(id)+"/send", nil, nil, "")
}

// UndoPendingPost turns an undoable post back into a draft.
func (c *Client) UndoPendingPost(ctx context.Context, id string) (Draft, *Response, error) {
	return doData[Draft](ctx, c, http.MethodPost, pendingPostEndpoint(id)+"/undo", nil, nil, "")
}

func doItemPage[T any](ctx context.Context, c *Client, endpoint, cursor string) ([]T, *Response, error) {
	query := url.Values{}
	setCursor(query, cursor)
	data, response, err := doData[struct {
		Items []T `json:"items"`
	}](ctx, c, http.MethodGet, endpoint, query, nil, "")

	return data.Items, response, err
}

func deleteSavedPosts(ctx context.Context, c *Client, endpoint string, ids []string) (int, *Response, error) {
	if len(ids) == 0 || len(ids) > 200 {
		return 0, nil, errors.New("north: delete requires between 1 and 200 IDs")
	}
	body, contentType, err := jsonRequest(struct {
		IDs []string `json:"ids"`
	}{IDs: ids})
	if err != nil {
		return 0, nil, err
	}
	data, response, err := doData[struct {
		Deleted int `json:"deleted"`
	}](ctx, c, http.MethodDelete, endpoint, nil, body, contentType)

	return data.Deleted, response, err
}

func pendingPostEndpoint(id string) string {
	return "/2/pending-posts/" + url.PathEscape(id)
}
