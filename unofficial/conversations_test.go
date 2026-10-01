package unofficial

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
)

func TestConversationEndpoints(t *testing.T) {
	t.Parallel()

	testEndpoints(t, []endpointTest{
		{"conversation", http.MethodGet, "/api/tweets/a%2Fb/conversation", url.Values{"cursor": {"next"}}, nil, func(ctx context.Context, client *Client) error {
			_, _, err := client.PostConversation(ctx, "a/b", "next")
			return err
		}},
		{"reply chain", http.MethodGet, "/api/tweets/a%2Fb/replies", nil, nil, func(ctx context.Context, client *Client) error {
			_, _, err := client.PostReplies(ctx, "a/b")
			return err
		}},
	})
}

func TestConversationTypesDecode(t *testing.T) {
	t.Parallel()

	var conversation PostConversationPage
	if err := json.Unmarshal([]byte(`{
		"ancestors":[{"id":"parent","media":[]}],
		"tweet":{"id":"root","media":[]},
		"replies":[{"id":"reply","media":[]}],
		"nextCursor":"next",
		"readerRootId":"parent"
	}`), &conversation); err != nil {
		t.Fatal(err)
	}
	if len(conversation.Ancestors) != 1 || conversation.Ancestors[0].ID != "parent" || conversation.Tweet.ID != "root" || len(conversation.Replies) != 1 || conversation.Replies[0].ID != "reply" {
		t.Fatalf("conversation = %#v", conversation)
	}
	if conversation.NextCursor == nil || *conversation.NextCursor != "next" || conversation.ReaderRootID == nil || *conversation.ReaderRootID != "parent" {
		t.Fatalf("conversation cursors = %#v", conversation)
	}

	var replies PostReplyPage
	if err := json.Unmarshal([]byte(`{"items":[{"id":"continued","media":[]}],"truncated":true}`), &replies); err != nil {
		t.Fatal(err)
	}
	if len(replies.Items) != 1 || replies.Items[0].ID != "continued" || !replies.Truncated {
		t.Fatalf("replies = %#v", replies)
	}
}
