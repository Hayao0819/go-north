package north

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"
)

const draftJSON = `{"id":"draft","items":[{"text":"hello","replyPolicy":"EVERYONE","media":[]}],"createdAt":"2026-10-01T00:00:00Z","updatedAt":"2026-10-01T00:00:00Z"}`

func TestDraftEndpoints(t *testing.T) {
	t.Parallel()

	seen := make(chan seenRequest, 16)
	client := newTestClient(t, func(writer http.ResponseWriter, request *http.Request) {
		seen <- seenRequest{method: request.Method, path: request.URL.Path, query: request.URL.Query()}
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/api/2/drafts":
			writeJSON(t, writer, http.StatusOK, `{"data":{"items":[`+draftJSON+`]}}`)
		case request.Method == http.MethodDelete:
			writeJSON(t, writer, http.StatusOK, `{"data":{"deleted":2}}`)
		case request.URL.Path == "/api/2/drafts" || request.URL.Path == "/api/2/drafts/draft":
			writeJSON(t, writer, http.StatusOK, `{"data":`+draftJSON+`}`)
		case request.Method == http.MethodGet && request.URL.Path == "/api/2/scheduled-posts":
			writeJSON(t, writer, http.StatusOK, `{"data":{"items":[`+draftJSON[:len(draftJSON)-1]+`,"scheduledAt":"2026-10-03T00:00:00Z","scheduleError":null}]}}`)
		case request.URL.Path == "/api/2/scheduled-posts" || request.URL.Path == "/api/2/scheduled-posts/scheduled":
			writeJSON(t, writer, http.StatusOK, `{"data":`+draftJSON[:len(draftJSON)-1]+`,"scheduledAt":"2026-10-03T00:00:00Z","scheduleError":null}}`)
		case request.Method == http.MethodGet && request.URL.Path == "/api/2/pending-posts":
			writeJSON(t, writer, http.StatusOK, `{"data":{"items":[`+draftJSON[:len(draftJSON)-1]+`,"undoUntil":"2026-10-02T00:00:30Z","scheduleError":null}]}}`)
		case request.URL.Path == "/api/2/pending-posts":
			writeJSON(t, writer, http.StatusCreated, `{"data":`+draftJSON[:len(draftJSON)-1]+`,"undoUntil":"2026-10-02T00:00:30Z","scheduleError":null}}`)
		case request.URL.Path == "/api/2/pending-posts/pending/send":
			writeJSON(t, writer, http.StatusAccepted, `{"data":{"sent":false,"pending":true}}`)
		case request.URL.Path == "/api/2/pending-posts/pending/undo":
			writeJSON(t, writer, http.StatusOK, `{"data":`+draftJSON+`}`)
		default:
			http.NotFound(writer, request)
		}
	})
	ctx := context.Background()
	request := DraftRequest{Items: []DraftItemInput{{Text: "hello", ReplyPolicy: ReplyEveryone}}}

	drafts, _, err := client.Drafts(ctx, "draft-cursor")
	if err != nil || len(drafts) != 1 {
		t.Fatalf("Drafts = %#v, %v", drafts, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/drafts", url.Values{"cursor": {"draft-cursor"}})
	if _, _, err := client.CreateDraft(ctx, request); err != nil {
		t.Fatal(err)
	}
	assertSeen(t, <-seen, http.MethodPost, "/api/2/drafts", nil)
	if _, _, err := client.UpdateDraft(ctx, "draft", request); err != nil {
		t.Fatal(err)
	}
	assertSeen(t, <-seen, http.MethodPatch, "/api/2/drafts/draft", nil)
	if deleted, _, err := client.DeleteDrafts(ctx, "one", "two"); err != nil || deleted != 2 {
		t.Fatalf("DeleteDrafts = %d, %v", deleted, err)
	}
	assertSeen(t, <-seen, http.MethodDelete, "/api/2/drafts", nil)

	scheduled, _, err := client.ScheduledPosts(ctx, "scheduled-cursor")
	if err != nil || len(scheduled) != 1 || scheduled[0].ID != "draft" {
		t.Fatalf("ScheduledPosts = %#v, %v", scheduled, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/scheduled-posts", url.Values{"cursor": {"scheduled-cursor"}})
	scheduleRequest := SchedulePostRequest{
		Item: DraftItemInput{Text: "later"}, ScheduledAt: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC),
	}
	if _, _, err := client.SchedulePost(ctx, scheduleRequest); err != nil {
		t.Fatal(err)
	}
	assertSeen(t, <-seen, http.MethodPost, "/api/2/scheduled-posts", nil)
	if _, _, err := client.UpdateScheduledPost(ctx, "scheduled", scheduleRequest); err != nil {
		t.Fatal(err)
	}
	assertSeen(t, <-seen, http.MethodPatch, "/api/2/scheduled-posts/scheduled", nil)
	if deleted, _, err := client.DeleteScheduledPosts(ctx, "one", "two"); err != nil || deleted != 2 {
		t.Fatalf("DeleteScheduledPosts = %d, %v", deleted, err)
	}
	assertSeen(t, <-seen, http.MethodDelete, "/api/2/scheduled-posts", nil)

	pending, _, err := client.PendingPosts(ctx, "pending-cursor")
	if err != nil || len(pending) != 1 || pending[0].ID != "draft" {
		t.Fatalf("PendingPosts = %#v, %v", pending, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/pending-posts", url.Values{"cursor": {"pending-cursor"}})
	if _, _, err := client.CreatePendingPost(ctx, CreatePendingPostRequest{Item: DraftItemInput{Text: "now"}}); err != nil {
		t.Fatal(err)
	}
	assertSeen(t, <-seen, http.MethodPost, "/api/2/pending-posts", nil)
	state, _, err := client.SendPendingPost(ctx, "pending")
	if err != nil || !state.Pending || state.Sent {
		t.Fatalf("SendPendingPost = %#v, %v", state, err)
	}
	assertSeen(t, <-seen, http.MethodPost, "/api/2/pending-posts/pending/send", nil)
	if _, _, err := client.UndoPendingPost(ctx, "pending"); err != nil {
		t.Fatal(err)
	}
	assertSeen(t, <-seen, http.MethodPost, "/api/2/pending-posts/pending/undo", nil)
}
