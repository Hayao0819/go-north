package north

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// BookmarkFolder groups bookmarked posts.
type BookmarkFolder struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	Count     int       `json:"count"`
	CreatedAt time.Time `json:"createdAt"`
}

// BookmarkPage is one page of bookmarks. Folder is set for folder pages.
type BookmarkPage struct {
	Items      []Post          `json:"items"`
	NextCursor *string         `json:"nextCursor"`
	Folder     *BookmarkFolder `json:"folder"`
}

// BookmarkState is returned after adding a bookmark.
type BookmarkState struct {
	Bookmarked bool `json:"bookmarked"`
	OK         bool `json:"ok"`
}

// Bookmarks returns the caller's bookmarks.
func (c *Client) Bookmarks(ctx context.Context, cursor string) (BookmarkPage, *Response, error) {
	query := url.Values{}
	setCursor(query, cursor)

	return doData[BookmarkPage](ctx, c, http.MethodGet, "/2/bookmarks", query, nil, "")
}

// ClearBookmarks removes every bookmark owned by the caller.
func (c *Client) ClearBookmarks(ctx context.Context) (int, *Response, error) {
	data, response, err := doData[struct {
		Deleted int `json:"deleted"`
	}](ctx, c, http.MethodDelete, "/2/bookmarks", nil, nil, "")

	return data.Deleted, response, err
}

// Bookmark adds a post to the caller's bookmarks.
func (c *Client) Bookmark(ctx context.Context, id string) (BookmarkState, *Response, error) {
	return doData[BookmarkState](ctx, c, http.MethodPost, postBookmarkEndpoint(id), nil, nil, "")
}

// Unbookmark removes a post from the caller's bookmarks.
func (c *Client) Unbookmark(ctx context.Context, id string) (bool, *Response, error) {
	return doOK(ctx, c, http.MethodDelete, postBookmarkEndpoint(id), nil, nil, "")
}

// BookmarkFolders returns the caller's bookmark folders.
func (c *Client) BookmarkFolders(ctx context.Context, cursor string) ([]BookmarkFolder, *Response, error) {
	query := url.Values{}
	setCursor(query, cursor)
	data, response, err := doData[struct {
		Items []BookmarkFolder `json:"items"`
	}](ctx, c, http.MethodGet, "/2/bookmark-folders", query, nil, "")

	return data.Items, response, err
}

// CreateBookmarkFolder creates a bookmark folder.
func (c *Client) CreateBookmarkFolder(ctx context.Context, name string) (BookmarkFolder, *Response, error) {
	return c.writeBookmarkFolder(ctx, http.MethodPost, "/2/bookmark-folders", name)
}

// UpdateBookmarkFolder renames a bookmark folder.
func (c *Client) UpdateBookmarkFolder(ctx context.Context, id, name string) (BookmarkFolder, *Response, error) {
	return c.writeBookmarkFolder(ctx, http.MethodPatch, "/2/bookmark-folders/"+url.PathEscape(id), name)
}

func (c *Client) writeBookmarkFolder(ctx context.Context, method, endpoint, name string) (BookmarkFolder, *Response, error) {
	body, contentType, err := jsonRequest(struct {
		Name string `json:"name"`
	}{Name: name})
	if err != nil {
		return BookmarkFolder{}, nil, err
	}

	return doData[BookmarkFolder](ctx, c, method, endpoint, nil, body, contentType)
}

// DeleteBookmarkFolder deletes a bookmark folder.
func (c *Client) DeleteBookmarkFolder(ctx context.Context, id string) (bool, *Response, error) {
	return doOK(ctx, c, http.MethodDelete, "/2/bookmark-folders/"+url.PathEscape(id), nil, nil, "")
}

// BookmarkFolderPosts returns one page from a bookmark folder.
func (c *Client) BookmarkFolderPosts(ctx context.Context, id, cursor string) (BookmarkPage, *Response, error) {
	query := url.Values{}
	setCursor(query, cursor)

	return doData[BookmarkPage](ctx, c, http.MethodGet, "/2/bookmark-folders/"+url.PathEscape(id)+"/bookmarks", query, nil, "")
}

// PostBookmarkFolders returns folders containing a post.
func (c *Client) PostBookmarkFolders(ctx context.Context, postID, cursor string) ([]BookmarkFolder, *Response, error) {
	query := url.Values{}
	setCursor(query, cursor)
	data, response, err := doData[struct {
		Items []BookmarkFolder `json:"items"`
	}](ctx, c, http.MethodGet, "/2/tweets/"+url.PathEscape(postID)+"/bookmark-folders", query, nil, "")

	return data.Items, response, err
}

// AddBookmarkToFolder adds a post to a bookmark folder.
func (c *Client) AddBookmarkToFolder(ctx context.Context, folderID, postID string) (BookmarkState, *Response, error) {
	return doData[BookmarkState](ctx, c, http.MethodPut, folderBookmarkEndpoint(folderID, postID), nil, nil, "")
}

// RemoveBookmarkFromFolder removes a post from a bookmark folder.
func (c *Client) RemoveBookmarkFromFolder(ctx context.Context, folderID, postID string) (bool, *Response, error) {
	return doOK(ctx, c, http.MethodDelete, folderBookmarkEndpoint(folderID, postID), nil, nil, "")
}

func postBookmarkEndpoint(id string) string {
	return "/2/tweets/" + url.PathEscape(id) + "/bookmark"
}

func folderBookmarkEndpoint(folderID, postID string) string {
	return "/2/bookmark-folders/" + url.PathEscape(folderID) + "/tweets/" + url.PathEscape(postID)
}
