package unofficial

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

func TestUserEndpoints(t *testing.T) {
	t.Parallel()

	testEndpoints(t, []endpointTest{
		{"user", http.MethodGet, "/api/users/alice", nil, nil, func(ctx context.Context, client *Client) error {
			_, _, err := client.User(ctx, "@alice")
			return err
		}},
		{"posts", http.MethodGet, "/api/users/alice/tweets", url.Values{"cursor": {"next"}, "tab": {"tweets"}}, nil, func(ctx context.Context, client *Client) error {
			_, _, err := client.UserPosts(ctx, "@alice", "next")
			return err
		}},
		{"mentions", http.MethodGet, "/api/search", url.Values{"cursor": {"next"}, "q": {"@alice"}, "tab": {"latest"}}, nil, func(ctx context.Context, client *Client) error {
			_, _, err := client.Mentions(ctx, "@alice", "next")
			return err
		}},
		{"update profile", http.MethodPatch, "/api/users/me", nil, map[string]any{"name": "Alice"}, func(ctx context.Context, client *Client) error {
			name := "Alice"
			_, err := client.UpdateProfile(ctx, UpdateProfileRequest{Name: &name})
			return err
		}},
	})
}
