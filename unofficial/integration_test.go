//go:build integration

package unofficial

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/Hayao0819/go-north"
)

func TestIntegrationWebReadOnly(t *testing.T) {
	cookie := os.Getenv("NORTH_SESSION_COOKIE")
	if cookie == "" {
		t.Skip("NORTH_SESSION_COOKIE is not set")
	}

	client, err := NewClient(cookie)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	user, response, err := client.Me(ctx)
	checkWebIntegrationResponse(t, "Me", response, err)
	if user.ID == "" || user.Handle == "" {
		t.Fatal("Me returned an incomplete account")
	}
	profile, response, err := client.User(ctx, user.Handle)
	checkWebIntegrationResponse(t, "User", response, err)
	if profile.ID != user.ID {
		t.Fatal("User returned a different account")
	}

	_, response, err = client.HomeTimeline(ctx, north.TimelineOptions{})
	checkWebIntegrationResponse(t, "HomeTimeline", response, err)

	_, response, err = client.SearchPosts(ctx, "from:"+user.Handle, north.SearchOptions{Tab: north.SearchLatest})
	checkWebIntegrationResponse(t, "SearchPosts", response, err)
}

func TestIntegrationPublicTrends(t *testing.T) {
	client, err := NewPublicClient()
	if err != nil {
		t.Fatalf("NewPublicClient: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, response, err := client.Trends(ctx)
	checkWebIntegrationResponse(t, "Trends", response, err)
}

func checkWebIntegrationResponse(t *testing.T, method string, response *Response, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", method, err)
	}
	if response == nil || response.StatusCode != http.StatusOK {
		t.Fatalf("%s returned an unexpected response", method)
	}
}
