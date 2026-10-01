package unofficial

import (
	"context"
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
