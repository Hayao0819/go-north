package north

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

const dmConversationJSON = `{"id":"conversation","name":null,"group":false,"participants":[{"id":"u1","handle":"alice","name":"Alice"}],"unreadCount":1,"request":false,"updatedAt":"2026-10-01T00:00:00Z"}`
const dmMessageJSON = `{"id":"message","conversationId":"conversation","sender":{"id":"u1","handle":"alice","name":"Alice"},"text":"hello","createdAt":"2026-10-01T00:00:00Z","read":false,"media":[],"reactions":[]}`

func TestDirectMessageEndpoints(t *testing.T) {
	t.Parallel()

	seen := make(chan seenRequest, 24)
	client := newTestClient(t, func(writer http.ResponseWriter, request *http.Request) {
		seen <- seenRequest{method: request.Method, path: request.URL.Path, query: request.URL.Query()}
		switch {
		case request.URL.Path == "/api/2/dm/conversations" && request.Method == http.MethodGet:
			writeJSON(t, writer, http.StatusOK, `{"data":{"items":[`+dmConversationJSON+`],"nextCursor":null,"requestCount":2}}`)
		case request.URL.Path == "/api/2/dm/conversations" && request.Method == http.MethodPost:
			writeJSON(t, writer, http.StatusCreated, `{"data":`+dmConversationJSON+`}`)
		case request.URL.Path == "/api/2/dm/conversations/conversation/messages" && request.Method == http.MethodGet:
			writeJSON(t, writer, http.StatusOK, `{"data":{"items":[`+dmMessageJSON+`],"nextCursor":null,"conversation":`+dmConversationJSON+`}}`)
		case request.URL.Path == "/api/2/dm/conversations/conversation/messages" && request.Method == http.MethodPost:
			writeJSON(t, writer, http.StatusCreated, `{"data":`+dmMessageJSON+`}`)
		case request.Method == http.MethodPatch && request.URL.Path == "/api/2/dm/conversations/conversation/messages/message":
			writeJSON(t, writer, http.StatusOK, `{"data":`+dmMessageJSON+`}`)
		case request.URL.Path == "/api/2/dm/conversations/conversation/messages/message/reaction":
			writeJSON(t, writer, http.StatusOK, `{"data":{"reactions":[{"emoji":"👍","count":1,"reactedByViewer":true,"users":[]}]}}`)
		case request.URL.Path == "/api/2/dm/unread-count":
			writeJSON(t, writer, http.StatusOK, `{"data":{"count":3}}`)
		case request.URL.Path == "/api/2/dm/recipients":
			writeJSON(t, writer, http.StatusOK, `{"data":{"items":[{"id":"u1","handle":"alice","name":"Alice"}],"nextCursor":null}}`)
		default:
			writeJSON(t, writer, http.StatusOK, `{"data":{"ok":true}}`)
		}
	})
	ctx := context.Background()

	conversations, _, err := client.DMConversations(ctx, "next", true)
	if err != nil || len(conversations.Items) != 1 || conversations.RequestCount != 2 {
		t.Fatalf("DMConversations = %#v, %v", conversations, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/dm/conversations", url.Values{"cursor": {"next"}, "requests": {"true"}})
	if _, _, err := client.CreateDMConversation(ctx, "alice"); err != nil {
		t.Fatal(err)
	}
	assertSeen(t, <-seen, http.MethodPost, "/api/2/dm/conversations", nil)
	if _, _, err := client.RenameDMConversation(ctx, "conversation", "Group"); err != nil {
		t.Fatal(err)
	}
	assertSeen(t, <-seen, http.MethodPatch, "/api/2/dm/conversations/conversation", nil)
	if _, _, err := client.AddDMConversationMembers(ctx, "conversation", "bob"); err != nil {
		t.Fatal(err)
	}
	assertSeen(t, <-seen, http.MethodPost, "/api/2/dm/conversations/conversation/members", nil)
	if _, _, err := client.LeaveDMConversation(ctx, "conversation"); err != nil {
		t.Fatal(err)
	}
	assertSeen(t, <-seen, http.MethodDelete, "/api/2/dm/conversations/conversation/members/me", nil)

	messages, _, err := client.DMMessages(ctx, "conversation", "older")
	if err != nil || len(messages.Items) != 1 || messages.Conversation.ID != "conversation" {
		t.Fatalf("DMMessages = %#v, %v", messages, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/dm/conversations/conversation/messages", url.Values{"cursor": {"older"}})
	message, _, err := client.SendDM(ctx, "conversation", SendDMRequest{Text: "hello"})
	if err != nil || message.ID != "message" {
		t.Fatalf("SendDM = %#v, %v", message, err)
	}
	assertSeen(t, <-seen, http.MethodPost, "/api/2/dm/conversations/conversation/messages", nil)
	if _, _, err := client.EditDM(ctx, "conversation", "message", "edited"); err != nil {
		t.Fatal(err)
	}
	assertSeen(t, <-seen, http.MethodPatch, "/api/2/dm/conversations/conversation/messages/message", nil)
	if _, _, err := client.DeleteDM(ctx, "conversation", "message", true); err != nil {
		t.Fatal(err)
	}
	assertSeen(t, <-seen, http.MethodDelete, "/api/2/dm/conversations/conversation/messages/message", url.Values{"scope": {"everyone"}})

	reactions, _, err := client.SetDMReaction(ctx, "conversation", "message", "👍")
	if err != nil || len(reactions) != 1 || !reactions[0].ReactedByViewer {
		t.Fatalf("SetDMReaction = %#v, %v", reactions, err)
	}
	assertSeen(t, <-seen, http.MethodPut, "/api/2/dm/conversations/conversation/messages/message/reaction", nil)
	if _, _, err := client.RemoveDMReaction(ctx, "conversation", "message"); err != nil {
		t.Fatal(err)
	}
	assertSeen(t, <-seen, http.MethodDelete, "/api/2/dm/conversations/conversation/messages/message/reaction", nil)

	writes := []struct {
		method string
		path   string
		call   func() (bool, *Response, error)
	}{
		{http.MethodPost, "/api/2/dm/conversations/conversation/read", func() (bool, *Response, error) { return client.MarkDMRead(ctx, "conversation") }},
		{http.MethodPost, "/api/2/dm/conversations/conversation/accept", func() (bool, *Response, error) { return client.AcceptDMRequest(ctx, "conversation") }},
		{http.MethodDelete, "/api/2/dm/conversations/conversation/request", func() (bool, *Response, error) { return client.DeleteDMRequest(ctx, "conversation") }},
	}
	for _, test := range writes {
		ok, _, err := test.call()
		if err != nil || !ok {
			t.Fatalf("%s = %v, %v", test.path, ok, err)
		}
		assertSeen(t, <-seen, test.method, test.path, nil)
	}

	if count, _, err := client.DMUnreadCount(ctx); err != nil || count != 3 {
		t.Fatalf("DMUnreadCount = %d, %v", count, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/dm/unread-count", nil)
	if users, _, err := client.DMRecipients(ctx, "ali", "users"); err != nil || len(users.Items) != 1 {
		t.Fatalf("DMRecipients = %#v, %v", users, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/dm/recipients", url.Values{"cursor": {"users"}, "q": {"ali"}})
}

func TestCreateDMConversationRequiresRecipient(t *testing.T) {
	t.Parallel()

	client, err := NewClient("token")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := client.CreateDMConversation(context.Background()); err == nil {
		t.Fatal("CreateDMConversation accepted no recipients")
	}
}
