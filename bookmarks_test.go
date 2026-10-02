package north

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

func TestBookmarkEndpoints(t *testing.T) {
	t.Parallel()

	seen := make(chan seenRequest, 20)
	client := newTestClient(t, func(writer http.ResponseWriter, request *http.Request) {
		seen <- seenRequest{method: request.Method, path: request.URL.Path, query: request.URL.Query()}
		switch {
		case request.URL.Path == "/api/2/bookmarks" && request.Method == http.MethodGet:
			writeJSON(t, writer, http.StatusOK, postListEnvelope("bookmark"))
		case request.URL.Path == "/api/2/bookmarks" && request.Method == http.MethodDelete:
			writeJSON(t, writer, http.StatusOK, `{"data":{"deleted":4}}`)
		case request.URL.Path == "/api/2/bookmark-folders" && request.Method == http.MethodGet:
			writeJSON(t, writer, http.StatusOK, `{"data":{"items":[{"id":"folder","name":"Read later","color":"blue"}]}}`)
		case request.URL.Path == "/api/2/bookmark-folders/folder/bookmarks":
			writeJSON(t, writer, http.StatusOK, `{"data":{"items":[`+postJSON("folder-post")+`],"nextCursor":null,"folder":{"id":"folder","name":"Read later","color":"blue"}}}`)
		case request.URL.Path == "/api/2/tweets/post/bookmark-folders":
			writeJSON(t, writer, http.StatusOK, `{"data":{"items":[{"id":"folder","name":"Read later","color":"blue"}]}}`)
		case request.Method == http.MethodPost && request.URL.Path == "/api/2/tweets/post/bookmark":
			writeJSON(t, writer, http.StatusOK, `{"data":{"bookmarked":true,"ok":true}}`)
		case request.Method == http.MethodPost || request.Method == http.MethodPatch:
			writeJSON(t, writer, http.StatusCreated, `{"data":{"id":"folder","name":"Read later","color":"blue"}}`)
		case request.Method == http.MethodPut:
			writeJSON(t, writer, http.StatusOK, `{"data":{"bookmarked":true,"ok":true}}`)
		default:
			writeJSON(t, writer, http.StatusOK, `{"data":{"ok":true}}`)
		}
	})
	ctx := context.Background()

	page, _, err := client.Bookmarks(ctx, "next")
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("Bookmarks = %#v, %v", page, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/bookmarks", url.Values{"cursor": {"next"}})

	deleted, _, err := client.ClearBookmarks(ctx)
	if err != nil || deleted != 4 {
		t.Fatalf("ClearBookmarks = %d, %v", deleted, err)
	}
	assertSeen(t, <-seen, http.MethodDelete, "/api/2/bookmarks", nil)

	state, _, err := client.Bookmark(ctx, "post")
	if err != nil || !state.Bookmarked {
		t.Fatalf("Bookmark = %#v, %v", state, err)
	}
	assertSeen(t, <-seen, http.MethodPost, "/api/2/tweets/post/bookmark", nil)
	if ok, _, err := client.Unbookmark(ctx, "post"); err != nil || !ok {
		t.Fatalf("Unbookmark = %v, %v", ok, err)
	}
	assertSeen(t, <-seen, http.MethodDelete, "/api/2/tweets/post/bookmark", nil)

	folders, _, err := client.BookmarkFolders(ctx, "folders")
	if err != nil || len(folders) != 1 {
		t.Fatalf("BookmarkFolders = %#v, %v", folders, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/bookmark-folders", url.Values{"cursor": {"folders"}})

	if _, _, err := client.CreateBookmarkFolder(ctx, "Read later"); err != nil {
		t.Fatal(err)
	}
	assertSeen(t, <-seen, http.MethodPost, "/api/2/bookmark-folders", nil)
	if _, _, err := client.UpdateBookmarkFolder(ctx, "folder", "Read later"); err != nil {
		t.Fatal(err)
	}
	assertSeen(t, <-seen, http.MethodPatch, "/api/2/bookmark-folders/folder", nil)
	if ok, _, err := client.DeleteBookmarkFolder(ctx, "folder"); err != nil || !ok {
		t.Fatalf("DeleteBookmarkFolder = %v, %v", ok, err)
	}
	assertSeen(t, <-seen, http.MethodDelete, "/api/2/bookmark-folders/folder", nil)

	page, _, err = client.BookmarkFolderPosts(ctx, "folder", "older")
	if err != nil || page.Folder == nil || page.Folder.ID != "folder" {
		t.Fatalf("BookmarkFolderPosts = %#v, %v", page, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/bookmark-folders/folder/bookmarks", url.Values{"cursor": {"older"}})
	folders, _, err = client.PostBookmarkFolders(ctx, "post", "cursor")
	if err != nil || len(folders) != 1 {
		t.Fatalf("PostBookmarkFolders = %#v, %v", folders, err)
	}
	assertSeen(t, <-seen, http.MethodGet, "/api/2/tweets/post/bookmark-folders", url.Values{"cursor": {"cursor"}})

	if _, _, err := client.AddBookmarkToFolder(ctx, "folder", "post"); err != nil {
		t.Fatal(err)
	}
	assertSeen(t, <-seen, http.MethodPut, "/api/2/bookmark-folders/folder/tweets/post", nil)
	if ok, _, err := client.RemoveBookmarkFromFolder(ctx, "folder", "post"); err != nil || !ok {
		t.Fatalf("RemoveBookmarkFromFolder = %v, %v", ok, err)
	}
	assertSeen(t, <-seen, http.MethodDelete, "/api/2/bookmark-folders/folder/tweets/post", nil)
}
