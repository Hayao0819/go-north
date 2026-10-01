package unofficial

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/Hayao0819/go-north"
)

func TestPostEndpoints(t *testing.T) {
	t.Parallel()

	testEndpoints(t, []endpointTest{
		{"post", http.MethodGet, "/api/tweets/a%2Fb", nil, nil, func(ctx context.Context, client *Client) error {
			_, _, err := client.Post(ctx, "a/b")
			return err
		}},
		{"home timeline", http.MethodGet, "/api/timeline/home", url.Values{"ranked": {"1"}}, nil, func(ctx context.Context, client *Client) error {
			_, _, err := client.HomeTimeline(ctx, north.TimelineOptions{Ranked: true})
			return err
		}},
		{"search", http.MethodGet, "/api/search", url.Values{"cursor": {"next"}, "q": {"from:alice"}, "tab": {"latest"}}, nil, func(ctx context.Context, client *Client) error {
			_, _, err := client.SearchPosts(ctx, "from:alice", north.SearchOptions{Tab: north.SearchLatest, Cursor: "next"})
			return err
		}},
		{"create post", http.MethodPost, "/api/tweets", nil, map[string]any{"text": "hello", "inReplyToId": "parent"}, func(ctx context.Context, client *Client) error {
			_, _, err := client.CreatePost(ctx, CreatePostRequest{Text: "hello", InReplyToID: "parent"})
			return err
		}},
		{"create poll", http.MethodPost, "/api/tweets", nil, map[string]any{"text": "choose", "poll": map[string]any{"options": []any{"one", "two"}, "durationMinutes": float64(60)}}, func(ctx context.Context, client *Client) error {
			_, _, err := client.CreatePost(ctx, CreatePostRequest{Text: "choose", Poll: &CreatePollRequest{Options: []string{"one", "two"}, DurationMinutes: 60}})
			return err
		}},
		{"delete post", http.MethodDelete, "/api/tweets/1", nil, nil, func(ctx context.Context, client *Client) error {
			_, _, err := client.DeletePost(ctx, "1")
			return err
		}},
		{"like", http.MethodPost, "/api/tweets/1/like", nil, nil, func(ctx context.Context, client *Client) error {
			_, _, err := client.Like(ctx, "1")
			return err
		}},
		{"unlike", http.MethodDelete, "/api/tweets/1/like", nil, nil, func(ctx context.Context, client *Client) error {
			_, _, err := client.Unlike(ctx, "1")
			return err
		}},
		{"repost", http.MethodPost, "/api/tweets/1/retweet", nil, nil, func(ctx context.Context, client *Client) error {
			_, _, err := client.Repost(ctx, "1")
			return err
		}},
		{"undo repost", http.MethodDelete, "/api/tweets/1/retweet", nil, nil, func(ctx context.Context, client *Client) error {
			_, _, err := client.UndoRepost(ctx, "1")
			return err
		}},
		{"edit post", http.MethodPut, "/api/tweets/1/edit", nil, map[string]any{"text": "edited", "mediaIds": []any{}}, func(ctx context.Context, client *Client) error {
			_, err := client.EditPost(ctx, "1", EditPostRequest{Text: "edited"})
			return err
		}},
		{"edit history", http.MethodGet, "/api/tweets/1/edit-history", nil, nil, func(ctx context.Context, client *Client) error {
			_, _, err := client.PostEditHistory(ctx, "1")
			return err
		}},
		{"vote poll", http.MethodPost, "/api/tweets/1/poll/vote", nil, map[string]any{"optionId": "option"}, func(ctx context.Context, client *Client) error {
			_, _, err := client.VotePoll(ctx, "1", "option")
			return err
		}},
	})
}

func TestPostDecodesPublicAndWebFields(t *testing.T) {
	t.Parallel()

	var post Post
	if err := json.Unmarshal([]byte(`{
		"id":"1",
		"text":"hello",
		"author":{"id":"u","handle":"alice"},
		"media":[{"id":"m1","kind":"PHOTO","status":"READY","url":"/media/1.jpg","errorMessage":"retrying","position":2}],
		"poll":{"id":"poll","endsAt":"2026-10-02T00:00:00Z","ended":false,"totalVotes":5,"viewerOptionId":"o2","options":[{"id":"o1","label":"one","position":0,"voteCount":2,"percent":40},{"id":"o2","label":"two","position":1,"voteCount":3,"percent":60}]},
		"quoted":{"id":"2","text":"quoted","author":{"id":"v","handle":"bob"},"media":[],"linkPreview":{"url":"https://example.com","title":null,"description":"example","imageUrl":null,"domain":"example.com","card":"summary"}},
		"retweetOf":{"id":"3","text":"original","author":{"id":"w","handle":"carol"},"media":[]},
		"editEligible":true,
		"pinned":true,
		"threadContinuation":[{"id":"4","text":"continued","author":{"id":"u","handle":"alice"},"media":[]}]
	}`), &post); err != nil {
		t.Fatal(err)
	}
	if post.ID != "1" || post.Text != "hello" || post.Author.Handle != "alice" || !post.EditEligible || !post.Pinned {
		t.Fatalf("post = %#v", post)
	}
	if len(post.Media) != 1 || post.Media[0].Position != 2 || post.Media[0].ErrorMessage == nil || *post.Media[0].ErrorMessage != "retrying" {
		t.Fatalf("media = %#v", post.Media)
	}
	if post.Poll == nil || post.Poll.TotalVotes != 5 || post.Poll.ViewerOptionID == nil || *post.Poll.ViewerOptionID != "o2" {
		t.Fatalf("poll = %#v", post.Poll)
	}
	if post.Quoted == nil || post.Quoted.LinkPreview == nil || post.Quoted.LinkPreview.Title != nil || post.Quoted.LinkPreview.Description == nil {
		t.Fatalf("quoted post = %#v", post.Quoted)
	}
	if post.DisplayPost() != post.RepostOf || len(post.ThreadContinuation) != 1 || post.ThreadContinuation[0].ID != "4" {
		t.Fatalf("post relationships = %#v", post)
	}

	public := post.PublicPost()
	if len(public.Media) != 1 || public.Media[0].ID != "m1" {
		t.Fatalf("public media = %#v", public.Media)
	}
	if public.Poll == nil || public.Poll.Voted == nil || *public.Poll.Voted != 1 || public.Poll.Options[1].Votes != 3 {
		t.Fatalf("public poll = %#v", public.Poll)
	}
	if public.Quoted == nil || public.Quoted.ID != "2" || public.RepostOf == nil || public.RepostOf.ID != "3" {
		t.Fatalf("public post relationships = %#v", public)
	}
}

func TestPostPageDecodesNewestAt(t *testing.T) {
	t.Parallel()

	var page PostPage
	if err := json.Unmarshal([]byte(`{"items":[{"id":"1","media":[]}],"nextCursor":"next","newestAt":"2026-10-01T12:00:00Z"}`), &page); err != nil {
		t.Fatal(err)
	}
	wantNewest := time.Date(2026, time.October, 1, 12, 0, 0, 0, time.UTC)
	if page.NewestAt == nil || !page.NewestAt.Equal(wantNewest) {
		t.Fatalf("newestAt = %v", page.NewestAt)
	}
	public := page.PublicPostPage()
	if len(public.Items) != 1 || public.Items[0].ID != "1" || public.NextCursor == nil || *public.NextCursor != "next" {
		t.Fatalf("public page = %#v", public)
	}
}
