package north

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// MuteDuration is the lifetime of a muted keyword.
type MuteDuration string

const (
	MuteForever MuteDuration = "forever"
	Mute24Hours MuteDuration = "24h"
	Mute7Days   MuteDuration = "7d"
	Mute30Days  MuteDuration = "30d"
)

// MutedKeyword is one keyword hidden by the caller.
type MutedKeyword struct {
	ID            string     `json:"id"`
	Word          string     `json:"word"`
	Home          bool       `json:"home"`
	Notifications bool       `json:"notifications"`
	FromAnyone    bool       `json:"fromAnyone"`
	ExpiresAt     *time.Time `json:"expiresAt"`
}

// CreateMutedKeywordRequest describes a new muted keyword.
type CreateMutedKeywordRequest struct {
	Word          string       `json:"word"`
	Home          bool         `json:"home"`
	Notifications bool         `json:"notifications"`
	FromAnyone    bool         `json:"fromAnyone"`
	Duration      MuteDuration `json:"duration"`
}

// MutedKeywords returns the caller's muted keywords.
func (c *Client) MutedKeywords(ctx context.Context, cursor string) ([]MutedKeyword, *Response, error) {
	query := url.Values{}
	setCursor(query, cursor)
	data, response, err := doData[struct {
		Items []MutedKeyword `json:"items"`
	}](ctx, c, http.MethodGet, "/2/muted-keywords", query, nil, "")

	return data.Items, response, err
}

// CreateMutedKeyword adds a muted keyword.
func (c *Client) CreateMutedKeyword(ctx context.Context, req CreateMutedKeywordRequest) (MutedKeyword, *Response, error) {
	body, contentType, err := jsonRequest(req)
	if err != nil {
		return MutedKeyword{}, nil, err
	}

	return doData[MutedKeyword](ctx, c, http.MethodPost, "/2/muted-keywords", nil, body, contentType)
}

// DeleteMutedKeyword removes a muted keyword.
func (c *Client) DeleteMutedKeyword(ctx context.Context, id string) (bool, *Response, error) {
	return doOK(ctx, c, http.MethodDelete, "/2/muted-keywords/"+url.PathEscape(id), nil, nil, "")
}
