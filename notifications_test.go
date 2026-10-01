package north

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNotificationEndpoints(t *testing.T) {
	t.Parallel()

	seen := make(chan seenRequest, 4)
	client := newTestClient(t, func(writer http.ResponseWriter, request *http.Request) {
		seen <- seenRequest{method: request.Method, path: request.URL.Path, query: request.URL.Query()}
		switch request.URL.Path {
		case "/api/2/notifications":
			body := `{"data":{"items":[{"id":"notice-1","kind":"LIKE","actors":[{"id":"user-1","handle":"alice","name":"Alice"}],"actorCount":2,"groupCount":3,"targetCount":1,"tweet":` + postJSON("post-1") + `,"createdAt":"2026-10-01T12:00:00Z","read":false}],"nextCursor":"older"}}`
			writeJSON(t, writer, http.StatusOK, body)
		case "/api/2/notifications/unread_count":
			writeJSON(t, writer, http.StatusOK, `{"data":{"count":4}}`)
		case "/api/2/notifications/read":
			writeJSON(t, writer, http.StatusOK, `{"data":{"marked":3}}`)
		default:
			http.NotFound(writer, request)
		}
	})

	ctx := context.Background()
	page, _, err := client.Notifications(ctx, NotificationsVerified, "next")
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.NextCursor == nil || *page.NextCursor != "older" {
		t.Fatalf("Notifications = %#v", page)
	}
	item := page.Items[0]
	if item.Kind != NotificationLike || item.ActorCount != 2 || item.GroupCount != 3 || item.TargetCount != 1 {
		t.Fatalf("notification counts = %#v", item)
	}
	if len(item.Actors) != 1 || item.Actors[0].Handle != "alice" || item.Post == nil || item.Post.ID != "post-1" {
		t.Fatalf("notification content = %#v", item)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/notifications", url.Values{"cursor": {"next"}, "tab": {"verified"}})

	if _, _, err := client.Notifications(ctx, "", ""); err != nil {
		t.Fatal(err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/notifications", url.Values{"tab": {"all"}})

	count, _, err := client.NotificationUnreadCount(ctx)
	if err != nil || count != 4 {
		t.Fatalf("NotificationUnreadCount = %d, %v", count, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/notifications/unread_count", nil)

	marked, _, err := client.MarkNotificationsRead(ctx)
	if err != nil || marked != 3 {
		t.Fatalf("MarkNotificationsRead = %d, %v", marked, err)
	}
	assertSeen(t, <-seen, http.MethodPost, "/api/2/notifications/read", nil)
}

func TestNotificationStream(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/api/2/notifications/stream" {
			t.Errorf("request = %s %s", request.Method, request.URL.Path)
		}
		if request.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("Accept = %q", request.Header.Get("Accept"))
		}
		writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		writer.Header().Set("X-Rate-Limit-Remaining", "29")
		_, _ = io.WriteString(writer, ": .\nretry: 7000\n\nevent: ignored\ndata: {}\n\nevent: notification\ndata: {\"id\":\"notice-1\",\"kind\":\"REPLY\",\"tweetId\":\"post-1\",\"createdAt\":\"2026-10-01T12:00:00Z\"}\n\n")
	})

	stream, response, err := client.StreamNotifications(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	if response == nil || response.StatusCode != http.StatusOK || response.RateLimit.Remaining != 29 {
		t.Fatalf("response = %#v", response)
	}

	event, err := stream.Receive()
	if err != nil {
		t.Fatal(err)
	}
	if event.ID != "notice-1" || event.Kind != NotificationReply || event.PostID != "post-1" {
		t.Fatalf("event = %#v", event)
	}
	if want := time.Date(2026, time.October, 1, 12, 0, 0, 0, time.UTC); !event.CreatedAt.Equal(want) {
		t.Fatalf("created at = %v, want %v", event.CreatedAt, want)
	}
	if stream.RetryAfter() != 7*time.Second {
		t.Fatalf("retry after = %v", stream.RetryAfter())
	}
	if _, err := stream.Receive(); !errors.Is(err, io.EOF) {
		t.Fatalf("second Receive error = %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestNotificationStreamErrors(t *testing.T) {
	t.Parallel()

	t.Run("API response", func(t *testing.T) {
		t.Parallel()

		client := newTestClient(t, func(writer http.ResponseWriter, _ *http.Request) {
			writeJSON(t, writer, http.StatusUnauthorized, `{"errors":[{"code":32,"message":"API key required"}]}`)
		})
		stream, response, err := client.StreamNotifications(context.Background())
		if stream != nil || response == nil || response.StatusCode != http.StatusUnauthorized {
			t.Fatalf("StreamNotifications = %#v, %#v, %v", stream, response, err)
		}
		var apiError *APIError
		if !errors.As(err, &apiError) || !apiError.HasCode(32) {
			t.Fatalf("error = %#v", err)
		}
	})

	t.Run("content type", func(t *testing.T) {
		t.Parallel()

		client := newTestClient(t, func(writer http.ResponseWriter, _ *http.Request) {
			writeJSON(t, writer, http.StatusOK, `{}`)
		})
		stream, _, err := client.StreamNotifications(context.Background())
		if stream != nil || err == nil || !strings.Contains(err.Error(), "content type") {
			t.Fatalf("StreamNotifications = %#v, %v", stream, err)
		}
	})

	t.Run("event data", func(t *testing.T) {
		t.Parallel()

		client := newTestClient(t, func(writer http.ResponseWriter, _ *http.Request) {
			writer.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(writer, "event: notification\ndata: {bad json}\n\n")
		})
		stream, _, err := client.StreamNotifications(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		defer stream.Close()
		if _, err := stream.Receive(); err == nil || !strings.Contains(err.Error(), "decode notification event") {
			t.Fatalf("Receive error = %v", err)
		}
	})
}
