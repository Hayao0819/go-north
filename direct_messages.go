package north

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"
)

// DMConversation is a direct-message conversation.
type DMConversation struct {
	ID           string     `json:"id"`
	Name         *string    `json:"name"`
	Group        bool       `json:"group"`
	Participants []User     `json:"participants"`
	LastMessage  *DMMessage `json:"lastMessage"`
	UnreadCount  int        `json:"unreadCount"`
	Request      bool       `json:"request"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}

// DMConversationPage is one cursor-paginated conversation page.
type DMConversationPage struct {
	Items        []DMConversation `json:"items"`
	NextCursor   *string          `json:"nextCursor"`
	RequestCount int              `json:"requestCount"`
}

// DMMessage is one direct message.
type DMMessage struct {
	ID             string       `json:"id"`
	ConversationID string       `json:"conversationId"`
	Sender         User         `json:"sender"`
	Text           string       `json:"text"`
	CreatedAt      time.Time    `json:"createdAt"`
	EditedAt       *time.Time   `json:"editedAt"`
	Read           bool         `json:"read"`
	System         bool         `json:"system"`
	Media          []Media      `json:"media"`
	Reactions      []DMReaction `json:"reactions"`
	ReplyTo        *DMReply     `json:"replyTo"`
	Post           *Post        `json:"tweet"`
}

// DMReaction groups one emoji and the accounts that used it.
type DMReaction struct {
	Emoji           string `json:"emoji"`
	Count           int    `json:"count"`
	ReactedByViewer bool   `json:"reactedByViewer"`
	Users           []User `json:"users"`
}

// DMReply is the compact message embedded as a reply target.
type DMReply struct {
	ID       string `json:"id"`
	Deleted  bool   `json:"deleted"`
	Sender   *User  `json:"sender"`
	Text     string `json:"text"`
	HasMedia bool   `json:"hasMedia"`
}

// DMMessagePage contains messages and their conversation metadata.
type DMMessagePage struct {
	Items        []DMMessage    `json:"items"`
	NextCursor   *string        `json:"nextCursor"`
	Conversation DMConversation `json:"conversation"`
}

// SendDMRequest is the content of a new direct message.
type SendDMRequest struct {
	Text      string   `json:"text,omitempty"`
	MediaIDs  []string `json:"mediaIds,omitempty"`
	ReplyToID *string  `json:"replyToId,omitempty"`
}

// DMConversations returns normal conversations or message requests.
func (c *Client) DMConversations(ctx context.Context, cursor string, requests bool) (DMConversationPage, *Response, error) {
	query := url.Values{"requests": {"false"}}
	if requests {
		query.Set("requests", "true")
	}
	setCursor(query, cursor)

	return doData[DMConversationPage](ctx, c, http.MethodGet, "/2/dm/conversations", query, nil, "")
}

// CreateDMConversation starts a conversation with one or more accounts.
func (c *Client) CreateDMConversation(ctx context.Context, handles ...string) (DMConversation, *Response, error) {
	if len(handles) == 0 || len(handles) > 19 {
		return DMConversation{}, nil, errors.New("north: CreateDMConversation requires between 1 and 19 recipients")
	}
	request := any(struct {
		Handles []string `json:"handles"`
	}{Handles: handles})
	if len(handles) == 1 {
		request = struct {
			Handle string `json:"handle"`
		}{Handle: handles[0]}
	}
	body, contentType, err := jsonRequest(request)
	if err != nil {
		return DMConversation{}, nil, err
	}

	return doData[DMConversation](ctx, c, http.MethodPost, "/2/dm/conversations", nil, body, contentType)
}

// RenameDMConversation changes a group conversation's name.
func (c *Client) RenameDMConversation(ctx context.Context, conversationID, name string) (bool, *Response, error) {
	body, contentType, err := jsonRequest(struct {
		Name string `json:"name"`
	}{Name: name})
	if err != nil {
		return false, nil, err
	}

	return doOK(ctx, c, http.MethodPatch, dmConversationEndpoint(conversationID), nil, body, contentType)
}

// AddDMConversationMembers adds accounts to a group conversation.
func (c *Client) AddDMConversationMembers(ctx context.Context, conversationID string, handles ...string) (bool, *Response, error) {
	if len(handles) == 0 || len(handles) > 19 {
		return false, nil, errors.New("north: AddDMConversationMembers requires between 1 and 19 accounts")
	}
	body, contentType, err := jsonRequest(struct {
		Handles []string `json:"handles"`
	}{Handles: handles})
	if err != nil {
		return false, nil, err
	}

	return doOK(ctx, c, http.MethodPost, dmConversationEndpoint(conversationID)+"/members", nil, body, contentType)
}

// LeaveDMConversation leaves a group conversation.
func (c *Client) LeaveDMConversation(ctx context.Context, conversationID string) (bool, *Response, error) {
	return doOK(ctx, c, http.MethodDelete, dmConversationEndpoint(conversationID)+"/members/me", nil, nil, "")
}

// DMMessages returns one page of messages and its conversation metadata.
func (c *Client) DMMessages(ctx context.Context, conversationID, cursor string) (DMMessagePage, *Response, error) {
	query := url.Values{}
	setCursor(query, cursor)

	return doData[DMMessagePage](ctx, c, http.MethodGet, dmConversationEndpoint(conversationID)+"/messages", query, nil, "")
}

// SendDM sends a message to a conversation.
func (c *Client) SendDM(ctx context.Context, conversationID string, req SendDMRequest) (DMMessage, *Response, error) {
	body, contentType, err := jsonRequest(req)
	if err != nil {
		return DMMessage{}, nil, err
	}

	return doData[DMMessage](ctx, c, http.MethodPost, dmConversationEndpoint(conversationID)+"/messages", nil, body, contentType)
}

// EditDM changes the text of a sent message.
func (c *Client) EditDM(ctx context.Context, conversationID, messageID, text string) (DMMessage, *Response, error) {
	body, contentType, err := jsonRequest(struct {
		Text string `json:"text"`
	}{Text: text})
	if err != nil {
		return DMMessage{}, nil, err
	}

	return doData[DMMessage](ctx, c, http.MethodPatch, dmMessageEndpoint(conversationID, messageID), nil, body, contentType)
}

// DeleteDM deletes a message locally or for every participant.
func (c *Client) DeleteDM(ctx context.Context, conversationID, messageID string, everyone bool) (bool, *Response, error) {
	query := url.Values{}
	if everyone {
		query.Set("scope", "everyone")
	}

	return doOK(ctx, c, http.MethodDelete, dmMessageEndpoint(conversationID, messageID), query, nil, "")
}

// SetDMReaction adds or replaces the caller's message reaction.
func (c *Client) SetDMReaction(ctx context.Context, conversationID, messageID, emoji string) ([]DMReaction, *Response, error) {
	body, contentType, err := jsonRequest(struct {
		Emoji string `json:"emoji"`
	}{Emoji: emoji})
	if err != nil {
		return nil, nil, err
	}

	return c.writeDMReaction(ctx, http.MethodPut, conversationID, messageID, body, contentType)
}

// RemoveDMReaction removes the caller's reaction from a message.
func (c *Client) RemoveDMReaction(ctx context.Context, conversationID, messageID string) ([]DMReaction, *Response, error) {
	return c.writeDMReaction(ctx, http.MethodDelete, conversationID, messageID, nil, "")
}

func (c *Client) writeDMReaction(ctx context.Context, method, conversationID, messageID string, body io.Reader, contentType string) ([]DMReaction, *Response, error) {
	data, response, err := doData[struct {
		Reactions []DMReaction `json:"reactions"`
	}](ctx, c, method, dmMessageEndpoint(conversationID, messageID)+"/reaction", nil, body, contentType)

	return data.Reactions, response, err
}

// MarkDMRead marks a conversation as read.
func (c *Client) MarkDMRead(ctx context.Context, conversationID string) (bool, *Response, error) {
	return doOK(ctx, c, http.MethodPost, dmConversationEndpoint(conversationID)+"/read", nil, nil, "")
}

// AcceptDMRequest accepts a message request.
func (c *Client) AcceptDMRequest(ctx context.Context, conversationID string) (bool, *Response, error) {
	return doOK(ctx, c, http.MethodPost, dmConversationEndpoint(conversationID)+"/accept", nil, nil, "")
}

// DeleteDMRequest deletes a message request and its history.
func (c *Client) DeleteDMRequest(ctx context.Context, conversationID string) (bool, *Response, error) {
	return doOK(ctx, c, http.MethodDelete, dmConversationEndpoint(conversationID)+"/request", nil, nil, "")
}

// DMUnreadCount returns the number of unread direct messages.
func (c *Client) DMUnreadCount(ctx context.Context) (int, *Response, error) {
	data, response, err := doData[struct {
		Count int `json:"count"`
	}](ctx, c, http.MethodGet, "/2/dm/unread-count", nil, nil, "")

	return data.Count, response, err
}

// DMRecipients searches accounts that can receive a direct message.
func (c *Client) DMRecipients(ctx context.Context, queryText, cursor string) (UserPage, *Response, error) {
	query := url.Values{}
	if queryText != "" {
		query.Set("q", queryText)
	}
	setCursor(query, cursor)

	return doData[UserPage](ctx, c, http.MethodGet, "/2/dm/recipients", query, nil, "")
}

func dmConversationEndpoint(id string) string {
	return "/2/dm/conversations/" + url.PathEscape(id)
}

func dmMessageEndpoint(conversationID, messageID string) string {
	return dmConversationEndpoint(conversationID) + "/messages/" + url.PathEscape(messageID)
}
