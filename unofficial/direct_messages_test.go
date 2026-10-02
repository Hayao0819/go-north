package unofficial

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
)

func TestDirectMessageEndpoints(t *testing.T) {
	t.Parallel()

	testEndpoints(t, []endpointTest{
		{"conversations", http.MethodGet, "/api/dm/conversations", url.Values{"cursor": {"next"}, "requests": {"true"}}, nil, func(ctx context.Context, client *Client) error {
			_, _, err := client.DMConversations(ctx, "next", true)
			return err
		}},
		{"create conversation", http.MethodPost, "/api/dm/conversations", nil, map[string]any{"handle": "alice"}, func(ctx context.Context, client *Client) error {
			_, _, err := client.CreateDMConversation(ctx, "alice")
			return err
		}},
		{"create group conversation", http.MethodPost, "/api/dm/conversations", nil, map[string]any{"handles": []any{"alice", "bob"}}, func(ctx context.Context, client *Client) error {
			_, _, err := client.CreateDMConversation(ctx, "alice", "bob")
			return err
		}},
		{"unread count", http.MethodGet, "/api/dm/unread-count", nil, nil, func(ctx context.Context, client *Client) error {
			_, _, err := client.DMUnreadCount(ctx)
			return err
		}},
		{"messages", http.MethodGet, "/api/dm/conversations/c/messages", url.Values{"cursor": {"older"}}, nil, func(ctx context.Context, client *Client) error {
			_, _, err := client.DMMessages(ctx, "c", "older")
			return err
		}},
		{"send", http.MethodPost, "/api/dm/conversations/c/messages", nil, map[string]any{"text": "hello"}, func(ctx context.Context, client *Client) error {
			_, err := client.SendDM(ctx, "c", SendDMRequest{Text: "hello"})
			return err
		}},
		{"edit", http.MethodPatch, "/api/dm/conversations/c/messages/m", nil, map[string]any{"text": "fixed"}, func(ctx context.Context, client *Client) error {
			_, err := client.EditDM(ctx, "c", "m", "fixed")
			return err
		}},
		{"delete for everyone", http.MethodDelete, "/api/dm/conversations/c/messages/m", url.Values{"scope": {"everyone"}}, nil, func(ctx context.Context, client *Client) error {
			_, err := client.DeleteDM(ctx, "c", "m", true)
			return err
		}},
		{"mark read", http.MethodPost, "/api/dm/conversations/c%2Fd/read", nil, nil, func(ctx context.Context, client *Client) error {
			_, err := client.MarkDMRead(ctx, "c/d")
			return err
		}},
		{"accept request", http.MethodPost, "/api/dm/conversations/c%2Fd/accept", nil, nil, func(ctx context.Context, client *Client) error {
			_, err := client.AcceptDMRequest(ctx, "c/d")
			return err
		}},
		{"delete request", http.MethodDelete, "/api/dm/conversations/c%2Fd/request", nil, nil, func(ctx context.Context, client *Client) error {
			_, err := client.DeleteDMRequest(ctx, "c/d")
			return err
		}},
		{"set reaction", http.MethodPut, "/api/dm/conversations/c%2Fd/messages/m%2Fn/reaction", nil, map[string]any{"emoji": "👍"}, func(ctx context.Context, client *Client) error {
			_, err := client.SetDMReaction(ctx, "c/d", "m/n", "👍")
			return err
		}},
		{"remove reaction", http.MethodDelete, "/api/dm/conversations/c%2Fd/messages/m%2Fn/reaction", nil, nil, func(ctx context.Context, client *Client) error {
			_, err := client.RemoveDMReaction(ctx, "c/d", "m/n")
			return err
		}},
	})
}

func TestDMTypesDecodeWebUsers(t *testing.T) {
	t.Parallel()

	var page DMMessagePage
	if err := json.Unmarshal([]byte(`{
		"items":[{
			"id":"m1",
			"conversationId":"c1",
			"sender":{"id":"u1","handle":"alice","avatarOriginalUrl":"/media/alice.png"},
			"reactions":[{"emoji":"👍","users":[{"id":"u2","handle":"bob","headerVideoUrl":"/media/bob.webm"}],"count":1,"reactedByViewer":true}],
			"replyTo":{"id":"m0","sender":{"id":"u3","handle":"carol","headerPosterUrl":"/media/carol.png"},"text":"hello","hasMedia":false,"deleted":false}
		}],
		"conversation":{"id":"c1","participants":[{"id":"u4","handle":"dave","birthday":{"month":10,"day":2}}]}
	}`), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ConversationID != "c1" || page.Items[0].Sender.AvatarOriginalURL == nil {
		t.Fatalf("message = %#v", page.Items)
	}
	reactions := page.Items[0].Reactions
	if len(reactions) != 1 || !reactions[0].ReactedByViewer || reactions[0].Count != 1 || len(reactions[0].Users) != 1 || reactions[0].Users[0].HeaderVideoURL == nil {
		t.Fatalf("reactions = %#v", reactions)
	}
	reply := page.Items[0].ReplyTo
	if reply == nil || reply.Sender == nil || reply.Sender.HeaderPosterURL == nil {
		t.Fatalf("reply = %#v", reply)
	}
	if len(page.Conversation.Participants) != 1 || page.Conversation.Participants[0].Birthday == nil {
		t.Fatalf("conversation = %#v", page.Conversation)
	}
}

func TestCreateDMConversationRequiresARecipient(t *testing.T) {
	t.Parallel()

	client, err := NewClient("session=secret")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := client.CreateDMConversation(context.Background()); err == nil {
		t.Fatal("CreateDMConversation accepted an empty recipient list")
	}
}
