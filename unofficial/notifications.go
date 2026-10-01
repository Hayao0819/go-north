package unofficial

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/Hayao0819/go-north"
)

// NotificationTab selects a notification feed.
type NotificationTab string

const (
	// NotificationsAll includes every notification visible to the account.
	NotificationsAll NotificationTab = "all"
	// NotificationsVerified includes notifications from verified accounts.
	NotificationsVerified NotificationTab = "verified"
	// NotificationsMentions includes mention, reply, and quote posts.
	NotificationsMentions NotificationTab = "mentions"
)

// NotificationKind identifies why a notification was created.
type NotificationKind string

const (
	// NotificationFollow reports a new follower.
	NotificationFollow NotificationKind = "FOLLOW"
	// NotificationLike reports a like.
	NotificationLike NotificationKind = "LIKE"
	// NotificationRepost reports a repost.
	NotificationRepost NotificationKind = "RETWEET"
	// NotificationPost reports a new-post alert from a followed account.
	NotificationPost NotificationKind = "POST"
	// NotificationReply contains a reply post.
	NotificationReply NotificationKind = "REPLY"
	// NotificationQuote contains a quote post.
	NotificationQuote NotificationKind = "QUOTE"
	// NotificationMention contains a mention post.
	NotificationMention NotificationKind = "MENTION"
)

// Notification is one item in the web notification feed.
type Notification struct {
	ID          string           `json:"id"`
	Kind        NotificationKind `json:"kind"`
	Read        bool             `json:"read"`
	Actors      []north.User     `json:"actors"`
	ActorCount  int              `json:"actorCount"`
	TargetCount int              `json:"targetCount"`
	CreatedAt   time.Time        `json:"createdAt"`
	Post        *Post            `json:"tweet"`
}

// NotificationPage is one cursor-paginated notification page.
type NotificationPage struct {
	Items      []Notification `json:"items"`
	NextCursor *string        `json:"nextCursor"`
}

// NotificationGroupKind identifies the item type in a notification group.
type NotificationGroupKind string

const (
	// NotificationGroupPosts contains posts affected by a notification.
	NotificationGroupPosts NotificationGroupKind = "tweets"
	// NotificationGroupUsers contains accounts affected by a notification.
	NotificationGroupUsers NotificationGroupKind = "users"
)

// NotificationGroupPage expands a notification that combines several targets.
// The web API uses the same items field for both posts and accounts.
type NotificationGroupPage struct {
	Kind       NotificationGroupKind `json:"kind"`
	Title      *string               `json:"title"`
	Posts      []Post                `json:"-"`
	Users      []User                `json:"-"`
	NextCursor *string               `json:"nextCursor"`
	RawItems   json.RawMessage       `json:"items"`
}

// UnmarshalJSON decodes the items field according to the group kind.
func (p *NotificationGroupPage) UnmarshalJSON(data []byte) error {
	value := struct {
		Kind       NotificationGroupKind `json:"kind"`
		Title      *string               `json:"title"`
		Items      json.RawMessage       `json:"items"`
		NextCursor *string               `json:"nextCursor"`
	}{}
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}

	p.Kind = value.Kind
	p.Title = value.Title
	p.NextCursor = value.NextCursor
	p.Posts = nil
	p.Users = nil
	p.RawItems = append(p.RawItems[:0], value.Items...)
	if len(value.Items) == 0 {
		return nil
	}

	switch value.Kind {
	case NotificationGroupPosts:
		if err := json.Unmarshal(value.Items, &p.Posts); err != nil {
			return fmt.Errorf("decode notification group posts: %w", err)
		}
	case NotificationGroupUsers:
		if err := json.Unmarshal(value.Items, &p.Users); err != nil {
			return fmt.Errorf("decode notification group users: %w", err)
		}
	}

	return nil
}

// Notifications returns one page from a web notification tab.
func (c *Client) Notifications(ctx context.Context, tab NotificationTab, cursor string) (NotificationPage, *Response, error) {
	if tab == "" {
		tab = NotificationsAll
	}
	query := url.Values{"tab": {string(tab)}}
	setCursor(query, cursor)

	return doJSON[NotificationPage](ctx, c, http.MethodGet, "/api/notifications", query, nil)
}

// NotificationUnreadCount returns the current unread notification count.
func (c *Client) NotificationUnreadCount(ctx context.Context) (int, *Response, error) {
	data, response, err := doJSON[struct {
		Count int `json:"count"`
	}](ctx, c, http.MethodGet, "/api/notifications/unread-count", nil, nil)

	return data.Count, response, err
}

// MarkNotificationsRead marks the notification feed as read.
func (c *Client) MarkNotificationsRead(ctx context.Context) (*Response, error) {
	return doEmpty(ctx, c, http.MethodPost, "/api/notifications/read", nil, nil)
}

// NotificationGroup expands a grouped notification.
func (c *Client) NotificationGroup(ctx context.Context, id, cursor string) (NotificationGroupPage, *Response, error) {
	query := url.Values{}
	setCursor(query, cursor)
	endpoint := "/api/notifications/group/" + url.PathEscape(id)

	return doJSON[NotificationGroupPage](ctx, c, http.MethodGet, endpoint, query, nil)
}
