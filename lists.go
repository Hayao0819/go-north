package north

import (
	"context"
	"net/http"
	"net/url"
	"time"
)

// List is a north user list.
type List struct {
	ID               string     `json:"id"`
	Name             string     `json:"name"`
	Description      *string    `json:"description"`
	Private          bool       `json:"private"`
	BannerURL        *string    `json:"bannerUrl"`
	MemberCount      int        `json:"memberCount"`
	FollowerCount    int        `json:"followerCount"`
	Owner            User       `json:"owner"`
	OwnedByViewer    bool       `json:"ownedByViewer"`
	FollowedByViewer bool       `json:"followedByViewer"`
	PinnedByViewer   bool       `json:"pinnedByViewer"`
	CreatedAt        *time.Time `json:"createdAt"`
}

// ListCollection groups owned, followed, and pinned lists.
type ListCollection struct {
	Items    []List `json:"items"`
	Followed []List `json:"followed"`
	Pinned   []List `json:"pinned"`
}

// ListMembership is an owned list with one account's membership state.
type ListMembership struct {
	List
	Contains bool `json:"contains"`
}

// CreateListRequest describes a new list.
type CreateListRequest struct {
	Name          string  `json:"name"`
	Description   *string `json:"description,omitempty"`
	Private       bool    `json:"private"`
	BannerMediaID *string `json:"bannerMediaId,omitempty"`
}

// UpdateListRequest contains list fields to change.
type UpdateListRequest struct {
	Name          *string         `json:"name,omitempty"`
	Description   *NullableString `json:"description,omitempty"`
	Private       *bool           `json:"private,omitempty"`
	BannerMediaID *NullableString `json:"bannerMediaId,omitempty"`
}

// Lists returns the caller's lists.
func (c *Client) Lists(ctx context.Context, cursor string) (ListCollection, *Response, error) {
	query := url.Values{}
	setCursor(query, cursor)

	return doData[ListCollection](ctx, c, http.MethodGet, "/2/lists", query, nil, "")
}

// CreateList creates a list.
func (c *Client) CreateList(ctx context.Context, req CreateListRequest) (List, *Response, error) {
	return c.writeList(ctx, http.MethodPost, "/2/lists", req)
}

// List returns one list.
func (c *Client) List(ctx context.Context, id string) (List, *Response, error) {
	return doData[List](ctx, c, http.MethodGet, listEndpoint(id), nil, nil, "")
}

// UpdateList changes a list owned by the caller.
func (c *Client) UpdateList(ctx context.Context, id string, req UpdateListRequest) (List, *Response, error) {
	return c.writeList(ctx, http.MethodPatch, listEndpoint(id), req)
}

func (c *Client) writeList(ctx context.Context, method, endpoint string, req any) (List, *Response, error) {
	body, contentType, err := jsonRequest(req)
	if err != nil {
		return List{}, nil, err
	}

	return doData[List](ctx, c, method, endpoint, nil, body, contentType)
}

// DeleteList deletes a list owned by the caller.
func (c *Client) DeleteList(ctx context.Context, id string) (bool, *Response, error) {
	return doOK(ctx, c, http.MethodDelete, listEndpoint(id), nil, nil, "")
}

// ListMemberships returns the caller's lists and whether handle is a member.
func (c *Client) ListMemberships(ctx context.Context, handle, cursor string) ([]ListMembership, *Response, error) {
	query := url.Values{"handle": {handle}}
	setCursor(query, cursor)
	data, response, err := doData[struct {
		Items []ListMembership `json:"items"`
	}](ctx, c, http.MethodGet, "/2/lists/memberships", query, nil, "")

	return data.Items, response, err
}

// ListMembers returns one page of list members.
func (c *Client) ListMembers(ctx context.Context, id, cursor string) (UserPage, *Response, error) {
	query := url.Values{}
	setCursor(query, cursor)

	return doData[UserPage](ctx, c, http.MethodGet, listEndpoint(id)+"/members", query, nil, "")
}

// ListTimeline returns one page of posts from list members.
func (c *Client) ListTimeline(ctx context.Context, id, cursor string) (PostPage, *Response, error) {
	query := url.Values{}
	setCursor(query, cursor)

	return doData[PostPage](ctx, c, http.MethodGet, listEndpoint(id)+"/timeline", query, nil, "")
}

// AddListMember adds an account to a list.
func (c *Client) AddListMember(ctx context.Context, id, handle string) (bool, *Response, error) {
	return c.setListResource(ctx, listEndpoint(id)+"/members/"+escapedHandle(handle), true)
}

// RemoveListMember removes an account from a list.
func (c *Client) RemoveListMember(ctx context.Context, id, handle string) (bool, *Response, error) {
	return c.setListResource(ctx, listEndpoint(id)+"/members/"+escapedHandle(handle), false)
}

// FollowList follows a list.
func (c *Client) FollowList(ctx context.Context, id string) (bool, *Response, error) {
	return c.setListResource(ctx, listEndpoint(id)+"/follow", true)
}

// UnfollowList stops following a list.
func (c *Client) UnfollowList(ctx context.Context, id string) (bool, *Response, error) {
	return c.setListResource(ctx, listEndpoint(id)+"/follow", false)
}

// PinList pins a list to the caller's home navigation.
func (c *Client) PinList(ctx context.Context, id string) (bool, *Response, error) {
	return c.setListResource(ctx, listEndpoint(id)+"/pin", true)
}

// UnpinList removes a list from the caller's pinned lists.
func (c *Client) UnpinList(ctx context.Context, id string) (bool, *Response, error) {
	return c.setListResource(ctx, listEndpoint(id)+"/pin", false)
}

func (c *Client) setListResource(ctx context.Context, endpoint string, on bool) (bool, *Response, error) {
	method := http.MethodDelete
	if on {
		method = http.MethodPost
	}

	return doOK(ctx, c, method, endpoint, nil, nil, "")
}

func listEndpoint(id string) string {
	return "/2/lists/" + url.PathEscape(id)
}
