package north

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
)

type seenRequest struct {
	method string
	path   string
	query  url.Values
}

func TestPostsAndTimelineEndpoints(t *testing.T) {
	t.Parallel()

	seen := make(chan seenRequest, 16)

	client := newTestClient(t, func(writer http.ResponseWriter, request *http.Request) {
		seen <- seenRequest{method: request.Method, path: request.URL.Path, query: request.URL.Query()}

		switch request.URL.Path {
		case "/api/2/tweets/counts":
			writeJSON(t, writer, http.StatusOK, `{"data":{"count":42}}`)
		case "/api/2/tweets/123/like":
			writeJSON(t, writer, http.StatusOK, `{"data":{"liked":true,"likeCount":4}}`)
		case "/api/2/tweets/123/retweet":
			writeJSON(t, writer, http.StatusOK, `{"data":{"retweeted":true,"retweetCount":5}}`)
		case "/api/2/tweets/123":
			if request.Method == http.MethodDelete {
				writeJSON(t, writer, http.StatusOK, `{"data":{"deleted":true}}`)
			} else {
				writeJSON(t, writer, http.StatusOK, postEnvelope("123"))
			}
		case "/api/2/tweets":
			if request.Method == http.MethodPost {
				var body CreatePostRequest
				if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
					t.Errorf("decode create request: %v", err)
				}
				if body.Text != "hello" {
					t.Errorf("create text = %q", body.Text)
				}
				writeJSON(t, writer, http.StatusCreated, `{"data":{"id":"created","text":"hello"}}`)
			} else {
				writeJSON(t, writer, http.StatusOK, postListEnvelope("1"))
			}
		default:
			writeJSON(t, writer, http.StatusOK, postListEnvelope("page"))
		}
	})

	ctx := context.Background()

	posts, _, err := client.Posts(ctx, "1", "2")
	if err != nil || len(posts) != 1 {
		t.Fatalf("Posts = %#v, %v", posts, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/tweets", url.Values{"ids": {"1,2"}})

	created, _, err := client.CreatePost(ctx, CreatePostRequest{Text: "hello"})
	if err != nil || created.ID != "created" {
		t.Fatalf("CreatePost = %#v, %v", created, err)
	}
	assertSeen(t, <-seen, http.MethodPost, "/api/2/tweets", nil)

	page, _, err := client.SearchPosts(ctx, "north #go", SearchOptions{Tab: SearchTop, Cursor: "next"})
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("SearchPosts = %#v, %v", page, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/tweets/search", url.Values{"q": {"north #go"}, "tab": {"top"}, "cursor": {"next"}})

	count, _, err := client.CountPosts(ctx, "from:north")
	if err != nil || count != 42 {
		t.Fatalf("CountPosts = %d, %v", count, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/tweets/counts", url.Values{"q": {"from:north"}})

	post, _, err := client.Post(ctx, "123")
	if err != nil || post.ID != "123" {
		t.Fatalf("Post = %#v, %v", post, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/tweets/123", nil)

	deleted, _, err := client.DeletePost(ctx, "123")
	if err != nil || !deleted {
		t.Fatalf("DeletePost = %v, %v", deleted, err)
	}
	assertSeen(t, <-seen, http.MethodDelete, "/api/2/tweets/123", nil)

	_, _, err = client.Quotes(ctx, "123", "older")
	if err != nil {
		t.Fatalf("Quotes: %v", err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/tweets/123/quotes", url.Values{"cursor": {"older"}})

	_, _, err = client.HomeTimeline(ctx, TimelineOptions{Ranked: true, Cursor: "more"})
	if err != nil {
		t.Fatalf("HomeTimeline: %v", err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/timelines/home", url.Values{"ranked": {"1"}, "cursor": {"more"}})

	like, _, err := client.Like(ctx, "123")
	if err != nil || !like.Liked || like.LikeCount != 4 {
		t.Fatalf("Like = %#v, %v", like, err)
	}
	assertSeen(t, <-seen, http.MethodPost, "/api/2/tweets/123/like", nil)

	_, _, err = client.Unlike(ctx, "123")
	if err != nil {
		t.Fatalf("Unlike: %v", err)
	}
	assertSeen(t, <-seen, http.MethodDelete, "/api/2/tweets/123/like", nil)

	repost, _, err := client.Repost(ctx, "123")
	if err != nil || !repost.Reposted || repost.RepostCount != 5 {
		t.Fatalf("Repost = %#v, %v", repost, err)
	}
	assertSeen(t, <-seen, http.MethodPost, "/api/2/tweets/123/retweet", nil)

	_, _, err = client.UndoRepost(ctx, "123")
	if err != nil {
		t.Fatalf("UndoRepost: %v", err)
	}
	assertSeen(t, <-seen, http.MethodDelete, "/api/2/tweets/123/retweet", nil)
}

func TestPostsBatchLimit(t *testing.T) {
	t.Parallel()

	client, err := NewClient("token")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := client.Posts(context.Background()); err == nil {
		t.Fatal("empty Posts succeeded")
	}
	if _, _, err := client.Posts(context.Background(), make([]string, 101)...); err == nil {
		t.Fatal("101 Posts succeeded")
	}
}

func assertSeen(t *testing.T, request seenRequest, method, path string, query url.Values) {
	t.Helper()

	if request.method != method || request.path != path || request.query.Encode() != query.Encode() {
		t.Errorf("request = %s %s %v, want %s %s %v", request.method, request.path, request.query, method, path, query)
	}
}

func postEnvelope(id string) string {
	return `{"data":` + postJSON(id) + `}`
}

func postListEnvelope(id string) string {
	return `{"data":{"items":[` + postJSON(id) + `],"nextCursor":null}}`
}

func postJSON(id string) string {
	return `{"id":"` + id + `","text":"hello","createdAt":"2026-09-28T00:00:00Z","author":{"id":"u1","handle":"north","name":"North"},"likeCount":0,"retweetCount":0,"replyCount":0,"quoteCount":0,"media":[]}`
}
