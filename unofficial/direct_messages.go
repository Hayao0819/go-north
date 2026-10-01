package unofficial

import (
	"context"
	"errors"
	"net/http"
	"net/url"
)

// DMUnreadCount returns the number of unread direct messages.
func (c *Client) DMUnreadCount(ctx context.Context) (int, *Response, error) {
	data, response, err := doJSON[struct {
		Count int `json:"count"`
	}](ctx, c, http.MethodGet, "/api/dm/unread-count", nil, nil)

	return data.Count, response, err
}

// DMConversations returns normal conversations or message requests.
func (c *Client) DMConversations(ctx context.Context, cursor string, requests bool) (DMConversationPage, *Response, error) {
	query := url.Values{"requests": {"false"}}
	if requests {
		query.Set("requests", "true")
	}
	setCursor(query, cursor)

	return doJSON[DMConversationPage](ctx, c, http.MethodGet, "/api/dm/conversations", query, nil)
}

// CreateDMConversation starts a conversation with one or more accounts.
func (c *Client) CreateDMConversation(ctx context.Context, handles ...string) (DMConversation, *Response, error) {
	if len(handles) == 0 {
		return DMConversation{}, nil, errors.New("north unofficial: at least one DM recipient is required")
	}
	request := any(struct {
		Handles []string `json:"handles"`
	}{Handles: handles})
	if len(handles) == 1 {
		request = struct {
			Handle string `json:"handle"`
		}{Handle: handles[0]}
	}

	return doJSON[DMConversation](ctx, c, http.MethodPost, "/api/dm/conversations", nil, request)
}

// DMMessages returns one page of messages and its conversation metadata.
func (c *Client) DMMessages(ctx context.Context, conversationID, cursor string) (DMMessagePage, *Response, error) {
	query := url.Values{}
	setCursor(query, cursor)
	endpoint := "/api/dm/conversations/" + url.PathEscape(conversationID) + "/messages"

	return doJSON[DMMessagePage](ctx, c, http.MethodGet, endpoint, query, nil)
}

// SendDM sends a message to a conversation.
func (c *Client) SendDM(ctx context.Context, conversationID string, request SendDMRequest) (*Response, error) {
	endpoint := "/api/dm/conversations/" + url.PathEscape(conversationID) + "/messages"

	return doEmpty(ctx, c, http.MethodPost, endpoint, nil, request)
}

// EditDM changes the text of a sent message.
func (c *Client) EditDM(ctx context.Context, conversationID, messageID, text string) (*Response, error) {
	endpoint := dmMessageEndpoint(conversationID, messageID)
	request := struct {
		Text string `json:"text"`
	}{Text: text}

	return doEmpty(ctx, c, http.MethodPatch, endpoint, nil, request)
}

// DeleteDM deletes a message locally or for every participant.
func (c *Client) DeleteDM(ctx context.Context, conversationID, messageID string, everyone bool) (*Response, error) {
	query := url.Values{}
	if everyone {
		query.Set("scope", "everyone")
	}

	return doEmpty(ctx, c, http.MethodDelete, dmMessageEndpoint(conversationID, messageID), query, nil)
}

// MarkDMRead marks a conversation as read.
func (c *Client) MarkDMRead(ctx context.Context, conversationID string) (*Response, error) {
	endpoint := "/api/dm/conversations/" + url.PathEscape(conversationID) + "/read"

	return doEmpty(ctx, c, http.MethodPost, endpoint, nil, nil)
}

// AcceptDMRequest accepts a message request.
func (c *Client) AcceptDMRequest(ctx context.Context, conversationID string) (*Response, error) {
	endpoint := "/api/dm/conversations/" + url.PathEscape(conversationID) + "/accept"

	return doEmpty(ctx, c, http.MethodPost, endpoint, nil, nil)
}

// DeleteDMRequest deletes a message request and its history.
func (c *Client) DeleteDMRequest(ctx context.Context, conversationID string) (*Response, error) {
	endpoint := "/api/dm/conversations/" + url.PathEscape(conversationID) + "/request"

	return doEmpty(ctx, c, http.MethodDelete, endpoint, nil, nil)
}

// SetDMReaction adds or replaces the current account's message reaction.
func (c *Client) SetDMReaction(ctx context.Context, conversationID, messageID, emoji string) (*Response, error) {
	request := struct {
		Emoji string `json:"emoji"`
	}{Emoji: emoji}

	return doEmpty(ctx, c, http.MethodPut, dmMessageEndpoint(conversationID, messageID)+"/reaction", nil, request)
}

// RemoveDMReaction removes the current account's reaction from a message.
func (c *Client) RemoveDMReaction(ctx context.Context, conversationID, messageID string) (*Response, error) {
	return doEmpty(ctx, c, http.MethodDelete, dmMessageEndpoint(conversationID, messageID)+"/reaction", nil, nil)
}

func dmMessageEndpoint(conversationID, messageID string) string {
	return "/api/dm/conversations/" + url.PathEscape(conversationID) + "/messages/" + url.PathEscape(messageID)
}
