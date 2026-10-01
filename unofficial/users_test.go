package unofficial

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
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

func TestUserDecodesWebProfileFields(t *testing.T) {
	t.Parallel()

	var user User
	if err := json.Unmarshal([]byte(`{
		"id":"u1",
		"handle":"alice",
		"birthday":{"month":10,"day":1},
		"birthdayVisibility":{"monthDay":"FOLLOWERS","year":"SELF"},
		"pinnedTweet":{"id":"p1","media":[]},
		"followRequested":true,
		"accountNotifications":true,
		"retweetsHidden":true,
		"followersYouKnow":{"users":[{"id":"u2","handle":"bob"}],"count":3},
		"suspended":true,
		"suspension":{"since":"2026-09-01T00:00:00Z","reason":"policy"}
	}`), &user); err != nil {
		t.Fatal(err)
	}
	if user.ID != "u1" || user.Birthday == nil || user.Birthday.Year != nil || user.Birthday.Month != 10 || user.Birthday.Day != 1 {
		t.Fatalf("user birthday = %#v", user)
	}
	if user.BirthdayVisibility == nil || user.BirthdayVisibility.MonthDay != BirthdayVisibilityFollowers || user.BirthdayVisibility.Year != BirthdayVisibilitySelf {
		t.Fatalf("birthday visibility = %#v", user.BirthdayVisibility)
	}
	if user.PinnedPost == nil || user.PinnedPost.ID != "p1" || !user.FollowRequested || !user.AccountNotifications || !user.RetweetsHidden {
		t.Fatalf("profile state = %#v", user)
	}
	if user.FollowersYouKnow == nil || user.FollowersYouKnow.Count != 3 || len(user.FollowersYouKnow.Users) != 1 {
		t.Fatalf("followers you know = %#v", user.FollowersYouKnow)
	}
	if !user.Suspended || user.Suspension == nil || user.Suspension.Since == nil || user.Suspension.Reason == nil || *user.Suspension.Reason != "policy" {
		t.Fatalf("suspension = %#v", user.Suspension)
	}
	if user.User.ID != "u1" || user.User.Handle != "alice" {
		t.Fatalf("public user = %#v", user.User)
	}
}

func TestUpdateProfileNullableFields(t *testing.T) {
	t.Parallel()

	name := "Alice"
	var clear *string
	visibility := BirthdayVisibilityFollowers
	data, err := json.Marshal(UpdateProfileRequest{Name: &name, Bio: &clear, BirthdayMonthDayVisibility: &visibility})
	if err != nil {
		t.Fatal(err)
	}
	value := string(data)
	if !strings.Contains(value, `"name":"Alice"`) || !strings.Contains(value, `"bio":null`) || !strings.Contains(value, `"birthdayMonthDayVisibility":"FOLLOWERS"`) || strings.Contains(value, "location") {
		t.Fatalf("request = %s", data)
	}
}

func TestSuspensionTimeDecodes(t *testing.T) {
	t.Parallel()

	var suspension Suspension
	if err := json.Unmarshal([]byte(`{"since":"2026-09-01T00:00:00Z"}`), &suspension); err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC)
	if suspension.Since == nil || !suspension.Since.Equal(want) {
		t.Fatalf("since = %v", suspension.Since)
	}
}
