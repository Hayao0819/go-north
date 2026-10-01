package unofficial

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

func TestNotificationEndpoints(t *testing.T) {
	t.Parallel()

	testEndpoints(t, []endpointTest{
		{"list", http.MethodGet, "/api/notifications", url.Values{"cursor": {"next"}, "tab": {"mentions"}}, nil, func(ctx context.Context, client *Client) error {
			_, _, err := client.Notifications(ctx, NotificationsMentions, "next")
			return err
		}},
		{"unread count", http.MethodGet, "/api/notifications/unread-count", nil, nil, func(ctx context.Context, client *Client) error {
			_, _, err := client.NotificationUnreadCount(ctx)
			return err
		}},
		{"group", http.MethodGet, "/api/notifications/group/a%2Fb", url.Values{"cursor": {"next/page"}}, nil, func(ctx context.Context, client *Client) error {
			_, _, err := client.NotificationGroup(ctx, "a/b", "next/page")
			return err
		}},
		{"mark read", http.MethodPost, "/api/notifications/read", nil, nil, func(ctx context.Context, client *Client) error {
			_, err := client.MarkNotificationsRead(ctx)
			return err
		}},
	})
}
