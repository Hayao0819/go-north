package north

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
)

func TestTrendAndMutedKeywordEndpoints(t *testing.T) {
	t.Parallel()

	seen := make(chan seenRequest, 8)
	client := newTestClient(t, func(writer http.ResponseWriter, request *http.Request) {
		seen <- seenRequest{method: request.Method, path: request.URL.Path, query: request.URL.Query()}
		switch {
		case request.URL.Path == "/api/2/trends" && request.Method == http.MethodGet:
			writeJSON(t, writer, http.StatusOK, `{"data":{"items":[{"tag":"Go","count":100,"isHashtag":false}]}}`)
		case request.URL.Path == "/api/2/muted-keywords" && request.Method == http.MethodGet:
			writeJSON(t, writer, http.StatusOK, `{"data":{"items":[{"id":"mute","word":"spoiler","home":true,"notifications":false,"fromAnyone":true,"expiresAt":null}]}}`)
		case request.URL.Path == "/api/2/muted-keywords" && request.Method == http.MethodPost:
			var body CreateMutedKeywordRequest
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode keyword: %v", err)
			}
			if body.Duration != Mute7Days || body.Word != "spoiler" {
				t.Errorf("keyword body = %#v", body)
			}
			writeJSON(t, writer, http.StatusCreated, `{"data":{"id":"mute","word":"spoiler","home":true}}`)
		default:
			writeJSON(t, writer, http.StatusOK, `{"data":{"ok":true}}`)
		}
	})
	ctx := context.Background()

	trends, _, err := client.Trends(ctx, "trend-cursor")
	if err != nil || len(trends) != 1 || trends[0].Count != 100 {
		t.Fatalf("Trends = %#v, %v", trends, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/trends", url.Values{"cursor": {"trend-cursor"}})
	if ok, _, err := client.DismissTrend(ctx, "Go"); err != nil || !ok {
		t.Fatalf("DismissTrend = %v, %v", ok, err)
	}
	assertSeen(t, <-seen, http.MethodPost, "/api/2/trends/dismiss", nil)

	keywords, _, err := client.MutedKeywords(ctx, "keyword-cursor")
	if err != nil || len(keywords) != 1 || keywords[0].Word != "spoiler" {
		t.Fatalf("MutedKeywords = %#v, %v", keywords, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/muted-keywords", url.Values{"cursor": {"keyword-cursor"}})
	keyword, _, err := client.CreateMutedKeyword(ctx, CreateMutedKeywordRequest{
		Word: "spoiler", Home: true, Duration: Mute7Days,
	})
	if err != nil || keyword.ID != "mute" {
		t.Fatalf("CreateMutedKeyword = %#v, %v", keyword, err)
	}
	assertSeen(t, <-seen, http.MethodPost, "/api/2/muted-keywords", nil)
	if ok, _, err := client.DeleteMutedKeyword(ctx, "mute"); err != nil || !ok {
		t.Fatalf("DeleteMutedKeyword = %v, %v", ok, err)
	}
	assertSeen(t, <-seen, http.MethodDelete, "/api/2/muted-keywords/mute", nil)
}
