package north

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

func TestConversationEndpoints(t *testing.T) {
	t.Parallel()

	seen := make(chan seenRequest, 8)
	client := newTestClient(t, func(writer http.ResponseWriter, request *http.Request) {
		seen <- seenRequest{method: request.Method, path: request.URL.Path, query: request.URL.Query()}
		switch request.URL.Path {
		case "/api/2/tweets/1/conversation":
			writeJSON(t, writer, http.StatusOK, `{"data":{"ancestors":[],"tweet":`+postJSON("1")+`,"replies":[],"nextCursor":"next"}}`)
		case "/api/2/tweets/1/replies":
			writeJSON(t, writer, http.StatusOK, `{"data":{"items":[`+postJSON("2")+`],"nextCursor":null,"truncated":true}}`)
		case "/api/2/tweets/1/edit-history":
			writeJSON(t, writer, http.StatusOK, `{"data":{"tweet":`+postJSON("1")+`,"versions":[{"id":"old","text":"old text","media":[],"createdAt":"2026-10-01T00:00:00Z","current":false}]}}`)
		default:
			writeJSON(t, writer, http.StatusOK, `{"data":{"items":[{"id":"u1","handle":"alice","name":"Alice"}],"nextCursor":null}}`)
		}
	})
	ctx := context.Background()

	conversation, _, err := client.Conversation(ctx, "1", "c1")
	if err != nil || conversation.Post.ID != "1" || conversation.NextCursor == nil {
		t.Fatalf("Conversation = %#v, %v", conversation, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/tweets/1/conversation", url.Values{"cursor": {"c1"}})

	replies, _, err := client.Replies(ctx, "1", "c2")
	if err != nil || len(replies.Items) != 1 || !replies.Truncated {
		t.Fatalf("Replies = %#v, %v", replies, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/tweets/1/replies", url.Values{"cursor": {"c2"}})

	history, _, err := client.PostEditHistory(ctx, "1")
	if err != nil || len(history.Versions) != 1 || history.Versions[0].ID != "old" {
		t.Fatalf("PostEditHistory = %#v, %v", history, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/tweets/1/edit-history", nil)

	likes, _, err := client.PostLikes(ctx, "1", "c3")
	if err != nil || len(likes.Items) != 1 {
		t.Fatalf("PostLikes = %#v, %v", likes, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/tweets/1/likes", url.Values{"cursor": {"c3"}})

	reposts, _, err := client.PostReposts(ctx, "1", "")
	if err != nil || len(reposts.Items) != 1 {
		t.Fatalf("PostReposts = %#v, %v", reposts, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/tweets/1/retweets", nil)
}
