package north

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

const listJSON = `{"id":"list","name":"Friends","private":false,"owner":{"id":"u1","handle":"alice","name":"Alice"}}`

func TestListEndpoints(t *testing.T) {
	t.Parallel()

	seen := make(chan seenRequest, 24)
	client := newTestClient(t, func(writer http.ResponseWriter, request *http.Request) {
		seen <- seenRequest{method: request.Method, path: request.URL.Path, query: request.URL.Query()}
		switch {
		case request.URL.Path == "/api/2/lists" && request.Method == http.MethodGet:
			writeJSON(t, writer, http.StatusOK, `{"data":{"items":[`+listJSON+`],"followed":[],"pinned":[]}}`)
		case request.URL.Path == "/api/2/lists/memberships":
			writeJSON(t, writer, http.StatusOK, `{"data":{"items":[`+listJSON[:len(listJSON)-1]+`,"contains":true}]}}`)
		case request.URL.Path == "/api/2/lists/list/members":
			writeJSON(t, writer, http.StatusOK, `{"data":{"items":[{"id":"u1","handle":"alice","name":"Alice"}],"nextCursor":null}}`)
		case request.URL.Path == "/api/2/lists/list/timeline":
			writeJSON(t, writer, http.StatusOK, postListEnvelope("post"))
		case request.Method == http.MethodGet || request.Method == http.MethodPost && request.URL.Path == "/api/2/lists" || request.Method == http.MethodPatch:
			writeJSON(t, writer, http.StatusOK, `{"data":`+listJSON+`}`)
		default:
			writeJSON(t, writer, http.StatusOK, `{"data":{"ok":true}}`)
		}
	})
	ctx := context.Background()

	collection, _, err := client.Lists(ctx, "lists")
	if err != nil || len(collection.Items) != 1 {
		t.Fatalf("Lists = %#v, %v", collection, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/lists", url.Values{"cursor": {"lists"}})
	if _, _, err := client.CreateList(ctx, CreateListRequest{Name: "Friends"}); err != nil {
		t.Fatal(err)
	}
	assertSeen(t, <-seen, http.MethodPost, "/api/2/lists", nil)
	list, _, err := client.List(ctx, "list")
	if err != nil || list.ID != "list" {
		t.Fatalf("List = %#v, %v", list, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/lists/list", nil)
	name := "Close friends"
	if _, _, err := client.UpdateList(ctx, "list", UpdateListRequest{Name: &name, Description: NullString()}); err != nil {
		t.Fatal(err)
	}
	assertSeen(t, <-seen, http.MethodPatch, "/api/2/lists/list", nil)
	if ok, _, err := client.DeleteList(ctx, "list"); err != nil || !ok {
		t.Fatalf("DeleteList = %v, %v", ok, err)
	}
	assertSeen(t, <-seen, http.MethodDelete, "/api/2/lists/list", nil)

	memberships, _, err := client.ListMemberships(ctx, "@bob", "membership")
	if err != nil || len(memberships) != 1 || !memberships[0].Contains {
		t.Fatalf("ListMemberships = %#v, %v", memberships, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/lists/memberships", url.Values{"cursor": {"membership"}, "handle": {"@bob"}})
	if page, _, err := client.ListMembers(ctx, "list", "members"); err != nil || len(page.Items) != 1 {
		t.Fatalf("ListMembers = %#v, %v", page, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/lists/list/members", url.Values{"cursor": {"members"}})
	if page, _, err := client.ListTimeline(ctx, "list", "timeline"); err != nil || len(page.Items) != 1 {
		t.Fatalf("ListTimeline = %#v, %v", page, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/lists/list/timeline", url.Values{"cursor": {"timeline"}})

	writes := []struct {
		method string
		path   string
		call   func() (bool, *Response, error)
	}{
		{http.MethodPost, "/api/2/lists/list/members/bob", func() (bool, *Response, error) { return client.AddListMember(ctx, "list", "@bob") }},
		{http.MethodDelete, "/api/2/lists/list/members/bob", func() (bool, *Response, error) { return client.RemoveListMember(ctx, "list", "bob") }},
		{http.MethodPost, "/api/2/lists/list/follow", func() (bool, *Response, error) { return client.FollowList(ctx, "list") }},
		{http.MethodDelete, "/api/2/lists/list/follow", func() (bool, *Response, error) { return client.UnfollowList(ctx, "list") }},
		{http.MethodPost, "/api/2/lists/list/pin", func() (bool, *Response, error) { return client.PinList(ctx, "list") }},
		{http.MethodDelete, "/api/2/lists/list/pin", func() (bool, *Response, error) { return client.UnpinList(ctx, "list") }},
	}
	for _, test := range writes {
		ok, _, err := test.call()
		if err != nil || !ok {
			t.Fatalf("%s %s = %v, %v", test.method, test.path, ok, err)
		}
		assertSeen(t, <-seen, test.method, test.path, nil)
	}
}
