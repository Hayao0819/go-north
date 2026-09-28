package north

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

func TestUserEndpoints(t *testing.T) {
	t.Parallel()

	type request struct {
		method string
		path   string
		query  url.Values
	}
	seen := make(chan request, 20)

	client := newTestClient(t, func(writer http.ResponseWriter, incoming *http.Request) {
		seen <- request{incoming.Method, incoming.URL.EscapedPath(), incoming.URL.Query()}

		switch {
		case incoming.URL.Path == "/api/2/users":
			writeJSON(t, writer, http.StatusOK, `{"data":{"items":[{"id":"1","handle":"alice","name":"Alice"}]}}`)
		case incoming.URL.Path == "/api/2/users/me" || incoming.URL.Path == "/api/2/users/alice":
			writeJSON(t, writer, http.StatusOK, `{"data":{"id":"1","handle":"alice","name":"Alice"}}`)
		case incoming.URL.Path == "/api/2/users/alice/followers" || incoming.URL.Path == "/api/2/users/alice/following":
			writeJSON(t, writer, http.StatusOK, `{"data":{"items":[{"id":"1","handle":"alice","name":"Alice"}],"nextCursor":null}}`)
		case incoming.Method == http.MethodGet:
			writeJSON(t, writer, http.StatusOK, postListEnvelope("user-post"))
		default:
			writeJSON(t, writer, http.StatusOK, `{"data":{"ok":true}}`)
		}
	})

	ctx := context.Background()
	users, _, err := client.Users(ctx, "alice", "@bob")
	if err != nil || len(users) != 1 {
		t.Fatalf("Users = %#v, %v", users, err)
	}
	got := <-seen
	if got.path != "/api/2/users" || got.query.Get("handles") != "alice,@bob" {
		t.Errorf("Users request = %#v", got)
	}

	me, _, err := client.Me(ctx)
	if err != nil || me.Handle != "alice" {
		t.Fatalf("Me = %#v, %v", me, err)
	}
	<-seen

	user, _, err := client.User(ctx, "@alice")
	if err != nil || user.Name != "Alice" {
		t.Fatalf("User = %#v, %v", user, err)
	}
	<-seen

	postCalls := []struct {
		name string
		path string
		call func() error
	}{
		{"posts", "/api/2/users/alice/tweets", func() error { _, _, err := client.UserPosts(ctx, "alice", "c"); return err }},
		{"mentions", "/api/2/users/alice/mentions", func() error { _, _, err := client.Mentions(ctx, "alice", "c"); return err }},
		{"likes", "/api/2/users/alice/liked_tweets", func() error { _, _, err := client.LikedPosts(ctx, "alice", "c"); return err }},
	}
	for _, call := range postCalls {
		if err := call.call(); err != nil {
			t.Fatalf("%s: %v", call.name, err)
		}
		got := <-seen
		if got.path != call.path || got.query.Get("cursor") != "c" {
			t.Errorf("%s request = %#v", call.name, got)
		}
	}

	for _, call := range []struct {
		name string
		path string
		call func() error
	}{
		{"followers", "/api/2/users/alice/followers", func() error { _, _, err := client.Followers(ctx, "alice", "u"); return err }},
		{"following", "/api/2/users/alice/following", func() error { _, _, err := client.Following(ctx, "alice", "u"); return err }},
	} {
		if err := call.call(); err != nil {
			t.Fatalf("%s: %v", call.name, err)
		}
		got := <-seen
		if got.path != call.path || got.query.Get("cursor") != "u" {
			t.Errorf("%s request = %#v", call.name, got)
		}
	}

	relations := []struct {
		method string
		path   string
		call   func() (bool, *Response, error)
	}{
		{http.MethodPost, "/api/2/users/alice/follow", func() (bool, *Response, error) { return client.Follow(ctx, "alice") }},
		{http.MethodDelete, "/api/2/users/alice/follow", func() (bool, *Response, error) { return client.Unfollow(ctx, "alice") }},
		{http.MethodPost, "/api/2/users/alice/block", func() (bool, *Response, error) { return client.Block(ctx, "alice") }},
		{http.MethodDelete, "/api/2/users/alice/block", func() (bool, *Response, error) { return client.Unblock(ctx, "alice") }},
		{http.MethodPost, "/api/2/users/alice/mute", func() (bool, *Response, error) { return client.Mute(ctx, "alice") }},
		{http.MethodDelete, "/api/2/users/alice/mute", func() (bool, *Response, error) { return client.Unmute(ctx, "alice") }},
	}
	for _, relation := range relations {
		ok, _, err := relation.call()
		if err != nil || !ok {
			t.Fatalf("relation %s %s = %v, %v", relation.method, relation.path, ok, err)
		}
		got := <-seen
		if got.method != relation.method || got.path != relation.path {
			t.Errorf("relation request = %#v", got)
		}
	}
}

func TestUserHandleIsPathEscaped(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(writer http.ResponseWriter, request *http.Request) {
		if got := request.URL.EscapedPath(); got != "/api/2/users/a%2Fb" {
			t.Errorf("escaped path = %q", got)
		}
		writeJSON(t, writer, http.StatusOK, `{"data":{"id":"1","handle":"a/b","name":"test"}}`)
	})

	if _, _, err := client.User(context.Background(), "a/b"); err != nil {
		t.Fatal(err)
	}
}

func TestUsersBatchLimit(t *testing.T) {
	t.Parallel()

	client, err := NewClient("token")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := client.Users(context.Background()); err == nil {
		t.Fatal("empty Users succeeded")
	}
	if _, _, err := client.Users(context.Background(), make([]string, 101)...); err == nil {
		t.Fatal("101 Users succeeded")
	}
}
