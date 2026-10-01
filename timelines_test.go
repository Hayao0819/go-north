package north

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

func TestTimelineEndpoints(t *testing.T) {
	t.Parallel()

	seen := make(chan seenRequest, 1)
	client := newTestClient(t, func(writer http.ResponseWriter, request *http.Request) {
		seen <- seenRequest{method: request.Method, path: request.URL.Path, query: request.URL.Query()}
		writeJSON(t, writer, http.StatusOK, postListEnvelope("timeline"))
	})

	page, _, err := client.HomeTimeline(context.Background(), TimelineOptions{Ranked: true, Cursor: "more"})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("HomeTimeline = %#v, %v", page, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/timelines/home", url.Values{"ranked": {"1"}, "cursor": {"more"}})
}
