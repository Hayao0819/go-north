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

func TestPostEndpoints(t *testing.T) {
	t.Parallel()

	seen := make(chan seenRequest, 16)

	client := newTestClient(t, func(writer http.ResponseWriter, request *http.Request) {
		seen <- seenRequest{method: request.Method, path: request.URL.Path, query: request.URL.Query()}

		switch request.URL.Path {
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

func TestDisplayPost(t *testing.T) {
	t.Parallel()

	original := &Post{ID: "original"}
	repost := &Post{ID: "repost", RepostOf: original}

	if got := original.DisplayPost(); got != original {
		t.Errorf("normal DisplayPost = %p, want %p", got, original)
	}
	if got := repost.DisplayPost(); got != original {
		t.Errorf("repost DisplayPost = %p, want %p", got, original)
	}
	var missing *Post
	if got := missing.DisplayPost(); got != nil {
		t.Errorf("nil DisplayPost = %#v", got)
	}
}

func TestNewPostWrites(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/2/tweets/1/edit":
			if request.Header.Get("If-Match") != `"version-1"` {
				t.Errorf("If-Match = %q", request.Header.Get("If-Match"))
			}
			var body EditPostRequest
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.Text == nil || *body.Text != "edited" {
				t.Errorf("edit body = %#v", body)
			}
			writeJSON(t, writer, http.StatusOK, postEnvelope("1"))
		case "/api/2/tweets/1/poll/vote":
			var body struct {
				OptionID string `json:"optionId"`
			}
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body.OptionID != "option-2" {
				t.Errorf("option ID = %q", body.OptionID)
			}
			writeJSON(t, writer, http.StatusOK, postEnvelope("1"))
		case "/api/2/tweets/thread":
			var body struct {
				Items []ThreadItem `json:"items"`
			}
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if len(body.Items) != 2 || body.Items[1].Poll == nil {
				t.Errorf("thread body = %#v", body)
			}
			writeJSON(t, writer, http.StatusCreated, `{"data":{"items":[`+postJSON("1")+`,`+postJSON("2")+`]}}`)
		default:
			http.NotFound(writer, request)
		}
	})
	ctx := context.Background()
	text := "edited"
	post, _, err := client.EditPostIfMatch(ctx, "1", EditPostRequest{Text: &text}, `"version-1"`)
	if err != nil || post.ID != "1" {
		t.Fatalf("EditPostIfMatch = %#v, %v", post, err)
	}
	if _, _, err := client.VotePoll(ctx, "1", "option-2"); err != nil {
		t.Fatalf("VotePoll: %v", err)
	}
	items, _, err := client.CreateThread(ctx, []ThreadItem{
		{Text: "first"},
		{Text: "second", Poll: &CreatePoll{Options: []string{"yes", "no"}, DurationMinutes: 60}},
	})
	if err != nil || len(items) != 2 {
		t.Fatalf("CreateThread = %#v, %v", items, err)
	}
}

func TestPollDecoding(t *testing.T) {
	t.Parallel()

	var post Post
	err := json.Unmarshal([]byte(`{"poll":{"id":"poll-1","endsAt":"2026-10-03T00:00:00Z","ended":false,"totalVotes":3,"viewerOptionId":"two","options":[{"id":"one","label":"One","position":0,"voteCount":1,"percent":33.3},{"id":"two","label":"Two","position":1,"voteCount":2,"percent":66.7}]},"editEligible":true}`), &post)
	if err != nil {
		t.Fatal(err)
	}
	if post.Poll == nil || post.Poll.TotalVotes != 3 || post.Poll.Options[1].VoteCount != 2 || !post.EditEligible {
		t.Fatalf("post = %#v", post)
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
