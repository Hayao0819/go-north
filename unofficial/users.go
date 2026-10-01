package unofficial

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Hayao0819/go-north"
)

// Birthday is the date displayed on a profile. Year is nil when it is hidden.
type Birthday struct {
	Year  *int `json:"year,omitempty"`
	Month int  `json:"month"`
	Day   int  `json:"day"`
}

// BirthdayVisibility controls who can see part of a birthday.
type BirthdayVisibility string

const (
	BirthdayVisibilityPublic    BirthdayVisibility = "PUBLIC"
	BirthdayVisibilityFollowers BirthdayVisibility = "FOLLOWERS"
	BirthdayVisibilityFollowing BirthdayVisibility = "FOLLOWING"
	BirthdayVisibilityMutual    BirthdayVisibility = "MUTUAL"
	BirthdayVisibilitySelf      BirthdayVisibility = "SELF"
)

// BirthdayVisibilitySettings contains the visibility of each birthday part.
type BirthdayVisibilitySettings struct {
	MonthDay BirthdayVisibility `json:"monthDay"`
	Year     BirthdayVisibility `json:"year"`
}

// FollowersYouKnow summarizes accounts followed by the current account that
// also follow this profile.
type FollowersYouKnow struct {
	Users []north.User `json:"users"`
	Count int          `json:"count"`
}

// Suspension contains the available metadata for an account suspension.
type Suspension struct {
	Since  *time.Time `json:"since"`
	Reason *string    `json:"reason"`
}

// User adds profile state returned by the web API to the public user type.
type User struct {
	north.User
	Birthday             *Birthday                   `json:"birthday"`
	BirthdayVisibility   *BirthdayVisibilitySettings `json:"birthdayVisibility"`
	PinnedPost           *Post                       `json:"pinnedTweet"`
	FollowRequested      bool                        `json:"followRequested"`
	AccountNotifications bool                        `json:"accountNotifications"`
	RetweetsHidden       bool                        `json:"retweetsHidden"`
	FollowersYouKnow     *FollowersYouKnow           `json:"followersYouKnow"`
	Suspended            bool                        `json:"suspended"`
	Suspension           *Suspension                 `json:"suspension"`
}

// UpdateProfileRequest uses pointer fields to distinguish omitted values from
// updates. Nullable fields use two pointers: nil omits a field, while a pointer
// to nil sends JSON null and clears it.
type UpdateProfileRequest struct {
	Name                       *string             `json:"name,omitempty"`
	Bio                        **string            `json:"bio,omitempty"`
	Location                   **string            `json:"location,omitempty"`
	Website                    **string            `json:"website,omitempty"`
	Birthday                   **Birthday          `json:"birthday,omitempty"`
	BirthdayMonthDayVisibility *BirthdayVisibility `json:"birthdayMonthDayVisibility,omitempty"`
	BirthdayYearVisibility     *BirthdayVisibility `json:"birthdayYearVisibility,omitempty"`
	AvatarMediaID              **string            `json:"avatarMediaId,omitempty"`
	HeaderMediaID              **string            `json:"headerMediaId,omitempty"`
}

// User fetches a public account profile.
func (c *Client) User(ctx context.Context, handle string) (User, *Response, error) {
	return doJSON[User](ctx, c, http.MethodGet, userEndpoint(handle, ""), nil, nil)
}

// UserPosts returns posts from an account's profile.
func (c *Client) UserPosts(ctx context.Context, handle, cursor string) (PostPage, *Response, error) {
	query := url.Values{"tab": {"tweets"}}
	setCursor(query, cursor)

	return doJSON[PostPage](ctx, c, http.MethodGet, userEndpoint(handle, "tweets"), query, nil)
}

// Mentions returns the latest posts that mention an account.
func (c *Client) Mentions(ctx context.Context, handle, cursor string) (PostPage, *Response, error) {
	handle = strings.TrimPrefix(strings.TrimSpace(handle), "@")

	return c.SearchPosts(ctx, "@"+handle, north.SearchOptions{Tab: north.SearchLatest, Cursor: cursor})
}

// UpdateProfile changes fields on the current account's public profile.
func (c *Client) UpdateProfile(ctx context.Context, request UpdateProfileRequest) (*Response, error) {
	return doEmpty(ctx, c, http.MethodPatch, "/api/users/me", nil, request)
}

func userEndpoint(handle, resource string) string {
	handle = strings.TrimPrefix(strings.TrimSpace(handle), "@")
	endpoint := "/api/users/" + url.PathEscape(handle)
	if resource != "" {
		endpoint += "/" + resource
	}

	return endpoint
}
