package unofficial

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

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

func TestNotificationGroupDecodesItemsByKind(t *testing.T) {
	t.Parallel()

	var page NotificationGroupPage
	if err := json.Unmarshal([]byte(`{"kind":"tweets","title":"Posts","items":[{"id":"p1","media":[]}],"nextCursor":"next"}`), &page); err != nil {
		t.Fatal(err)
	}
	if page.Kind != NotificationGroupPosts || page.Title == nil || *page.Title != "Posts" || len(page.Posts) != 1 || page.Posts[0].ID != "p1" || len(page.Users) != 0 {
		t.Fatalf("post group = %#v", page)
	}

	if err := json.Unmarshal([]byte(`{"kind":"users","title":"People","items":[{"id":"u1","handle":"alice","birthday":{"year":2000,"month":1,"day":2}}],"nextCursor":null}`), &page); err != nil {
		t.Fatal(err)
	}
	if page.Kind != NotificationGroupUsers || len(page.Users) != 1 || page.Users[0].Handle != "alice" || len(page.Posts) != 0 {
		t.Fatalf("user group = %#v", page)
	}
	if page.Users[0].Birthday == nil || page.Users[0].Birthday.Year == nil || *page.Users[0].Birthday.Year != 2000 {
		t.Fatalf("group user birthday = %#v", page.Users[0].Birthday)
	}

	if err := json.Unmarshal([]byte(`{"kind":"future","title":"Future","items":[{"value":1}]}`), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Posts) != 0 || len(page.Users) != 0 || string(page.RawItems) != `[{"value":1}]` {
		t.Fatalf("unknown group = %#v", page)
	}
}

func TestDMReactionDecodesViewerState(t *testing.T) {
	t.Parallel()

	var reaction DMReaction
	if err := json.Unmarshal([]byte(`{"emoji":"👍","users":[{"id":"u1","handle":"alice"}],"count":1,"reactedByViewer":true}`), &reaction); err != nil {
		t.Fatal(err)
	}
	if !reaction.ReactedByViewer || reaction.Count != 1 || len(reaction.Users) != 1 {
		t.Fatalf("reaction = %#v", reaction)
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
