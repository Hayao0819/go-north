package unofficial

import (
	"context"
	"net/http"
	"net/url"
	"testing"

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
