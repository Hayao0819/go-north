//go:build integration

package north

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestIntegrationReadOnly(t *testing.T) {
	token := os.Getenv("NORTH_API_KEY")
	if token == "" {
		t.Skip("NORTH_API_KEY is not set")
	}

	client, err := NewClient(token)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	me, response, err := client.Me(ctx)
	checkIntegrationResponse(t, "Me", response, err)
	if me.ID == "" || me.Handle == "" {
		t.Fatal("Me returned an incomplete account")
	}

	_, response, err = client.HomeTimeline(ctx, TimelineOptions{})
	checkIntegrationResponse(t, "HomeTimeline", response, err)

	query := SearchQuery{From: me.Handle}.String()
	_, response, err = client.SearchPosts(ctx, query, SearchOptions{Tab: SearchLatest})
	checkIntegrationResponse(t, "SearchPosts", response, err)

	count, response, err := client.CountPosts(ctx, query)
	checkIntegrationResponse(t, "CountPosts", response, err)
	if count < 0 {
		t.Fatalf("CountPosts returned %d", count)
	}

	_, response, err = client.Notifications(ctx, NotificationsAll, "")
	checkIntegrationResponse(t, "Notifications", response, err)

	unread, response, err := client.NotificationUnreadCount(ctx)
	checkIntegrationResponse(t, "NotificationUnreadCount", response, err)
	if unread < 0 {
		t.Fatalf("NotificationUnreadCount returned %d", unread)
	}

	user, response, err := client.User(ctx, me.Handle)
	checkIntegrationResponse(t, "User", response, err)
	if user.ID != me.ID {
		t.Fatal("User returned a different account")
	}
}

func TestIntegrationDMReadOnly(t *testing.T) {
	token := os.Getenv("NORTH_API_KEY")
	if token == "" {
		t.Skip("NORTH_API_KEY is not set")
	}

	client, err := NewClient(token)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if client.TokenKind() == TokenLegacy {
		t.Skip("NORTH_API_KEY is a legacy token")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conversations, response, err := client.DMConversations(ctx, "", false)
	checkIntegrationResponse(t, "DMConversations", response, err)

	unread, response, err := client.DMUnreadCount(ctx)
	checkIntegrationResponse(t, "DMUnreadCount", response, err)
	if unread < 0 {
		t.Fatalf("DMUnreadCount returned %d", unread)
	}

	_, response, err = client.DMConversations(ctx, "", true)
	checkIntegrationResponse(t, "DMRequests", response, err)

	if len(conversations.Items) == 0 {
		return
	}
	conversation := conversations.Items[0]
	if conversation.ID == "" {
		t.Fatal("DMConversations returned an item without an ID")
	}
	page, response, err := client.DMMessages(ctx, conversation.ID, "")
	checkIntegrationResponse(t, "DMMessages", response, err)
	if page.Conversation.ID != conversation.ID {
		t.Fatal("DMMessages returned a different conversation")
	}
}

func checkIntegrationResponse(t *testing.T, method string, response *Response, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", method, err)
	}
	if response == nil || response.StatusCode != http.StatusOK {
		t.Fatalf("%s returned an unexpected response", method)
	}
}
