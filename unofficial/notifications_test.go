package unofficial

import (
	"context"
	"encoding/json"
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

func TestNotificationGroupDecodesItemsByKind(t *testing.T) {
	t.Parallel()

	var page NotificationGroupPage
	if err := json.Unmarshal([]byte(`{"kind":"tweets","title":"Posts","items":[{"id":"p1","media":[]}],"nextCursor":"next"}`), &page); err != nil {
		t.Fatal(err)
	}
	if page.Kind != NotificationGroupPosts || page.Title == nil || *page.Title != "Posts" || len(page.Posts) != 1 || page.Posts[0].ID != "p1" || len(page.Users) != 0 {
		t.Fatalf("post group = %#v", page)
	}

	if err := json.Unmarshal([]byte(`{"kind":"users","title":"People","items":[{"id":"u1","handle":"alice","birthday":{"year":2000,"month":1,"day":2}}],"nextCursor":null}`), &page); err != nil {
		t.Fatal(err)
	}
	if page.Kind != NotificationGroupUsers || len(page.Users) != 1 || page.Users[0].Handle != "alice" || len(page.Posts) != 0 {
		t.Fatalf("user group = %#v", page)
	}
	if page.Users[0].Birthday == nil || page.Users[0].Birthday.Year == nil || *page.Users[0].Birthday.Year != 2000 {
		t.Fatalf("group user birthday = %#v", page.Users[0].Birthday)
	}

	if err := json.Unmarshal([]byte(`{"kind":"future","title":"Future","items":[{"value":1}]}`), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Posts) != 0 || len(page.Users) != 0 || string(page.RawItems) != `[{"value":1}]` {
		t.Fatalf("unknown group = %#v", page)
	}
}

func TestNotificationDecodesWebActors(t *testing.T) {
	t.Parallel()

	var notification Notification
	if err := json.Unmarshal([]byte(`{"id":"n1","kind":"FOLLOW","actors":[{"id":"u1","handle":"alice","avatarOriginalUrl":"/media/alice.png"}],"createdAt":"2026-10-02T00:00:00Z"}`), &notification); err != nil {
		t.Fatal(err)
	}
	if len(notification.Actors) != 1 || notification.Actors[0].AvatarOriginalURL == nil || *notification.Actors[0].AvatarOriginalURL != "/media/alice.png" {
		t.Fatalf("actors = %#v", notification.Actors)
	}
}
