package north

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// User is a north account. Fields not included by a particular endpoint keep
// their zero value; nullable profile fields use pointers to preserve null.
type User struct {
	ID             string     `json:"id"`
	Handle         string     `json:"handle"`
	Name           string     `json:"name"`
	Bio            *string    `json:"bio"`
	Location       *string    `json:"location"`
	Website        *string    `json:"website"`
	AvatarURL      *string    `json:"avatarUrl"`
	HeaderURL      *string    `json:"headerUrl"`
	Protected      bool       `json:"protected"`
	Verified       bool       `json:"verified"`
	FollowerCount  int        `json:"followerCount"`
	FollowingCount int        `json:"followingCount"`
	PostCount      int        `json:"tweetCount"`
	CreatedAt      *time.Time `json:"createdAt"`
	Following      bool       `json:"following"`
	FollowedBy     bool       `json:"followedBy"`
	Blocking       bool       `json:"blocking"`
	Muting         bool       `json:"muting"`
}

// UserPage is one cursor-paginated page of accounts.
type UserPage struct {
	Items      []User  `json:"items"`
	NextCursor *string `json:"nextCursor"`
}

// NullableString distinguishes an omitted profile field from a JSON null.
// Use NewNullableString to set a value and NullString to clear one.
type NullableString struct {
	value *string
}

// NewNullableString returns a profile field containing value.
func NewNullableString(value string) *NullableString {
	return &NullableString{value: &value}
}

// NullString returns a profile field that is encoded as JSON null.
func NullString() *NullableString {
	return &NullableString{}
}

// MarshalJSON implements json.Marshaler.
func (s NullableString) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.value)
}

// UpdateProfileRequest contains the mutable account profile fields. Nil fields
// are not changed.
type UpdateProfileRequest struct {
	Name          *string         `json:"name,omitempty"`
	Bio           *NullableString `json:"bio,omitempty"`
	Location      *NullableString `json:"location,omitempty"`
	Website       *NullableString `json:"website,omitempty"`
	AvatarMediaID *NullableString `json:"avatarMediaId,omitempty"`
	HeaderMediaID *NullableString `json:"headerMediaId,omitempty"`
}

// Users fetches up to 100 accounts in the same order as handles. A leading @
// is accepted by the service.
func (c *Client) Users(ctx context.Context, handles ...string) ([]User, *Response, error) {
	if len(handles) == 0 || len(handles) > 100 {
		return nil, nil, errors.New("north: Users requires between 1 and 100 handles")
	}

	data, resp, err := doData[struct {
		Items []User `json:"items"`
	}](ctx, c, http.MethodGet, "/2/users", url.Values{"handles": {strings.Join(handles, ",")}}, nil, "")

	return data.Items, resp, err
}

// Me fetches the account that owns the API key.
func (c *Client) Me(ctx context.Context) (User, *Response, error) {
	return doData[User](ctx, c, http.MethodGet, "/2/users/me", nil, nil, "")
}

// UpdateProfile changes the caller's profile.
func (c *Client) UpdateProfile(ctx context.Context, req UpdateProfileRequest) (User, *Response, error) {
	body, contentType, err := jsonRequest(req)
	if err != nil {
		return User{}, nil, err
	}

	return doData[User](ctx, c, http.MethodPatch, "/2/users/me", nil, body, contentType)
}

// User fetches a public account profile.
func (c *Client) User(ctx context.Context, handle string) (User, *Response, error) {
	return doData[User](ctx, c, http.MethodGet, userEndpoint(handle, ""), nil, nil, "")
}

// UserPosts returns posts by an account.
func (c *Client) UserPosts(ctx context.Context, handle, cursor string) (PostPage, *Response, error) {
	return c.userPostPage(ctx, handle, "tweets", cursor)
}

// Mentions returns public posts containing @handle. It is not the private
// notification feed.
func (c *Client) Mentions(ctx context.Context, handle, cursor string) (PostPage, *Response, error) {
	return c.userPostPage(ctx, handle, "mentions", cursor)
}

// LikedPosts returns public posts liked by an account.
func (c *Client) LikedPosts(ctx context.Context, handle, cursor string) (PostPage, *Response, error) {
	return c.userPostPage(ctx, handle, "liked_tweets", cursor)
}

func (c *Client) userPostPage(ctx context.Context, handle, resource, cursor string) (PostPage, *Response, error) {
	q := url.Values{}
	setCursor(q, cursor)

	return doData[PostPage](ctx, c, http.MethodGet, userEndpoint(handle, resource), q, nil, "")
}

// Followers returns one page of an account's followers.
func (c *Client) Followers(ctx context.Context, handle, cursor string) (UserPage, *Response, error) {
	return c.userPage(ctx, handle, "followers", cursor)
}

// Following returns one page of accounts followed by handle.
func (c *Client) Following(ctx context.Context, handle, cursor string) (UserPage, *Response, error) {
	return c.userPage(ctx, handle, "following", cursor)
}

// BlockedUsers returns accounts blocked by the caller.
func (c *Client) BlockedUsers(ctx context.Context, cursor string) (UserPage, *Response, error) {
	return c.viewerUserPage(ctx, "blocked", cursor)
}

// MutedUsers returns accounts muted by the caller.
func (c *Client) MutedUsers(ctx context.Context, cursor string) (UserPage, *Response, error) {
	return c.viewerUserPage(ctx, "muted", cursor)
}

func (c *Client) viewerUserPage(ctx context.Context, resource, cursor string) (UserPage, *Response, error) {
	query := url.Values{}
	setCursor(query, cursor)

	return doData[UserPage](ctx, c, http.MethodGet, "/2/users/me/"+resource, query, nil, "")
}

// FollowRequests returns pending requests to follow the caller.
func (c *Client) FollowRequests(ctx context.Context, cursor string) (UserPage, *Response, error) {
	query := url.Values{}
	setCursor(query, cursor)

	return doData[UserPage](ctx, c, http.MethodGet, "/2/follow-requests", query, nil, "")
}

// AcceptFollowRequest accepts a pending follow request.
func (c *Client) AcceptFollowRequest(ctx context.Context, handle string) (bool, *Response, error) {
	return c.setFollowRequest(ctx, handle, true)
}

// RejectFollowRequest rejects a pending follow request.
func (c *Client) RejectFollowRequest(ctx context.Context, handle string) (bool, *Response, error) {
	return c.setFollowRequest(ctx, handle, false)
}

func (c *Client) setFollowRequest(ctx context.Context, handle string, accept bool) (bool, *Response, error) {
	method := http.MethodDelete
	if accept {
		method = http.MethodPost
	}
	data, response, err := doData[struct {
		OK bool `json:"ok"`
	}](ctx, c, method, "/2/follow-requests/"+escapedHandle(handle), nil, nil, "")

	return data.OK, response, err
}

func (c *Client) userPage(ctx context.Context, handle, resource, cursor string) (UserPage, *Response, error) {
	q := url.Values{}
	setCursor(q, cursor)

	return doData[UserPage](ctx, c, http.MethodGet, userEndpoint(handle, resource), q, nil, "")
}

// Follow follows an account. A protected account receives a follow request.
func (c *Client) Follow(ctx context.Context, handle string) (bool, *Response, error) {
	return c.setRelation(ctx, handle, "follow", true)
}

// Unfollow stops following an account.
func (c *Client) Unfollow(ctx context.Context, handle string) (bool, *Response, error) {
	return c.setRelation(ctx, handle, "follow", false)
}

// Block blocks an account and removes follows in both directions.
func (c *Client) Block(ctx context.Context, handle string) (bool, *Response, error) {
	return c.setRelation(ctx, handle, "block", true)
}

// Unblock removes an account block.
func (c *Client) Unblock(ctx context.Context, handle string) (bool, *Response, error) {
	return c.setRelation(ctx, handle, "block", false)
}

// Mute hides an account from timelines and notifications without unfollowing.
func (c *Client) Mute(ctx context.Context, handle string) (bool, *Response, error) {
	return c.setRelation(ctx, handle, "mute", true)
}

// Unmute removes an account mute.
func (c *Client) Unmute(ctx context.Context, handle string) (bool, *Response, error) {
	return c.setRelation(ctx, handle, "mute", false)
}

// EnablePostNotifications enables new-post notifications for an account.
func (c *Client) EnablePostNotifications(ctx context.Context, handle string) (bool, *Response, error) {
	return c.setPostNotifications(ctx, handle, true)
}

// DisablePostNotifications disables new-post notifications for an account.
func (c *Client) DisablePostNotifications(ctx context.Context, handle string) (bool, *Response, error) {
	return c.setPostNotifications(ctx, handle, false)
}

func (c *Client) setPostNotifications(ctx context.Context, handle string, on bool) (bool, *Response, error) {
	method := http.MethodDelete
	if on {
		method = http.MethodPost
	}
	data, response, err := doData[struct {
		OK bool `json:"ok"`
	}](ctx, c, method, userEndpoint(handle, "account-notifications"), nil, nil, "")

	return data.OK, response, err
}

func (c *Client) setRelation(ctx context.Context, handle, relation string, on bool) (bool, *Response, error) {
	method := http.MethodDelete
	if on {
		method = http.MethodPost
	}

	data, resp, err := doData[struct {
		OK bool `json:"ok"`
	}](ctx, c, method, userEndpoint(handle, relation), nil, nil, "")

	return data.OK, resp, err
}

func userEndpoint(handle, resource string) string {
	path := "/2/users/" + escapedHandle(handle)
	if resource != "" {
		path += "/" + resource
	}

	return path
}

func escapedHandle(handle string) string {
	return url.PathEscape(strings.TrimPrefix(handle, "@"))
}
