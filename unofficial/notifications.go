package unofficial

import (
	"context"
	"net/http"
	"net/url"
)

// Notifications returns one page from a web notification tab.
func (c *Client) Notifications(ctx context.Context, tab NotificationTab, cursor string) (NotificationPage, *Response, error) {
	if tab == "" {
		tab = NotificationsAll
	}
	query := url.Values{"tab": {string(tab)}}
	setCursor(query, cursor)

	return doJSON[NotificationPage](ctx, c, http.MethodGet, "/api/notifications", query, nil)
}

// NotificationUnreadCount returns the current unread notification count.
func (c *Client) NotificationUnreadCount(ctx context.Context) (int, *Response, error) {
	data, response, err := doJSON[struct {
		Count int `json:"count"`
	}](ctx, c, http.MethodGet, "/api/notifications/unread-count", nil, nil)

	return data.Count, response, err
}

// MarkNotificationsRead marks the notification feed as read.
func (c *Client) MarkNotificationsRead(ctx context.Context) (*Response, error) {
	return doEmpty(ctx, c, http.MethodPost, "/api/notifications/read", nil, nil)
}

// NotificationGroup expands a grouped notification.
func (c *Client) NotificationGroup(ctx context.Context, id, cursor string) (NotificationGroupPage, *Response, error) {
	query := url.Values{}
	setCursor(query, cursor)
	endpoint := "/api/notifications/group/" + url.PathEscape(id)

	return doJSON[NotificationGroupPage](ctx, c, http.MethodGet, endpoint, query, nil)
}
