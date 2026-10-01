package unofficial

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Hayao0819/go-north"
)

// LoginRequirements describes the browser verification required by north.
type LoginRequirements struct {
	TurnstileEnabled bool   `json:"enabled"`
	TurnstileSiteKey string `json:"siteKey"`
}

// LoginRequest contains credentials and browser verification for one login.
type LoginRequest struct {
	Identifier     string
	Password       string
	TurnstileToken string
	StartedAt      time.Time
	AddAccount     bool
}

// LoginResult reports whether the account requires a second login step.
type LoginResult struct {
	RequiresTwoFactor bool   `json:"requires2fa"`
	TwoFactorToken    string `json:"token"`
}

// Post adds web-only fields to the public API's post representation.
type Post struct {
	north.Post
	Media              []Media      `json:"media"`
	Poll               *Poll        `json:"poll"`
	Quoted             *Post        `json:"quoted"`
	RepostOf           *Post        `json:"retweetOf"`
	EditEligible       bool         `json:"editEligible"`
	Pinned             bool         `json:"pinned"`
	LinkPreview        *LinkPreview `json:"linkPreview"`
	ThreadContinuation []Post       `json:"threadContinuation"`
}

// DisplayPost returns the original post for a repost.
func (p *Post) DisplayPost() *Post {
	if p != nil && p.RepostOf != nil {
		return p.RepostOf
	}

	return p
}

// PublicPost returns the fields shared with the public API representation.
func (p Post) PublicPost() north.Post {
	result := p.Post
	if p.Media != nil {
		result.Media = make([]north.Media, len(p.Media))
		for index := range p.Media {
			result.Media[index] = p.Media[index].Media
		}
	}
	if p.Poll != nil {
		poll := p.Poll.publicPoll()
		result.Poll = &poll
	}
	if p.Quoted != nil {
		quoted := p.Quoted.PublicPost()
		result.Quoted = &quoted
	}
	if p.RepostOf != nil {
		repost := p.RepostOf.PublicPost()
		result.RepostOf = &repost
	}

	return result
}

// LinkPreviewCard controls how a link preview is displayed.
type LinkPreviewCard string

const (
	// LinkPreviewSummary displays a compact preview with a thumbnail.
	LinkPreviewSummary LinkPreviewCard = "summary"
	// LinkPreviewSummaryLargeImage displays a preview with a large image.
	LinkPreviewSummaryLargeImage LinkPreviewCard = "summary_large_image"
)

// LinkPreview contains metadata fetched for a URL in a post.
type LinkPreview struct {
	URL         string          `json:"url"`
	Title       *string         `json:"title"`
	Description *string         `json:"description"`
	ImageURL    *string         `json:"imageUrl"`
	Domain      string          `json:"domain"`
	Card        LinkPreviewCard `json:"card"`
}

// Media adds fields returned by the web API to the public media type.
type Media struct {
	north.Media
	ErrorMessage *string `json:"errorMessage"`
	Position     int     `json:"position"`
}

// Poll is a poll attached to a web API post.
type Poll struct {
	ID             string       `json:"id"`
	EndsAt         time.Time    `json:"endsAt"`
	Ended          bool         `json:"ended"`
	TotalVotes     int          `json:"totalVotes"`
	ViewerOptionID *string      `json:"viewerOptionId"`
	Options        []PollOption `json:"options"`
}

func (p Poll) publicPoll() north.Poll {
	result := north.Poll{EndsAt: p.EndsAt, Options: make([]north.PollOption, len(p.Options))}
	for index, option := range p.Options {
		result.Options[index] = north.PollOption{Label: option.Label, Votes: option.VoteCount}
		if p.ViewerOptionID != nil && option.ID == *p.ViewerOptionID {
			position := option.Position
			result.Voted = &position
		}
	}

	return result
}

// PollOption is one choice in a web API poll.
type PollOption struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Position  int    `json:"position"`
	VoteCount int    `json:"voteCount"`
	Percent   int    `json:"percent"`
}

// PostPage is one cursor-paginated page of posts.
type PostPage struct {
	Items      []Post     `json:"items"`
	NextCursor *string    `json:"nextCursor"`
	NewestAt   *time.Time `json:"newestAt"`
}

// PublicPostPage returns the fields shared with a public API post page.
func (p PostPage) PublicPostPage() north.PostPage {
	items := make([]north.Post, len(p.Items))
	for index := range p.Items {
		items[index] = p.Items[index].PublicPost()
	}

	return north.PostPage{Items: items, NextCursor: p.NextCursor}
}

// CreatePostRequest is the request used by the web client when publishing.
type CreatePostRequest struct {
	Text        string             `json:"text,omitempty"`
	ReplyPolicy north.ReplyPolicy  `json:"replyPolicy,omitempty"`
	MediaIDs    []string           `json:"mediaIds,omitempty"`
	InReplyToID string             `json:"inReplyToId,omitempty"`
	QuotedID    string             `json:"quotedId,omitempty"`
	Poll        *CreatePollRequest `json:"poll,omitempty"`
}

// CreatePollRequest contains the choices and duration of a new poll.
type CreatePollRequest struct {
	Options         []string `json:"options"`
	DurationMinutes int      `json:"durationMinutes"`
}

// EditPostRequest is the editable portion of a post.
type EditPostRequest struct {
	Text     string   `json:"text"`
	MediaIDs []string `json:"mediaIds"`
}

// PostEditHistory contains the current post and its retained versions.
type PostEditHistory struct {
	Post     Post              `json:"tweet"`
	Versions []PostEditVersion `json:"versions"`
}

// PostEditVersion is one entry in a post's edit history.
type PostEditVersion struct {
	ID        string    `json:"id"`
	Text      string    `json:"text"`
	Media     []Media   `json:"media"`
	CreatedAt time.Time `json:"createdAt"`
	Current   bool      `json:"current"`
}

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

// DMConversation describes a direct-message conversation.
type DMConversation struct {
	ID           string       `json:"id"`
	Name         *string      `json:"name"`
	Group        bool         `json:"group"`
	Request      bool         `json:"request"`
	Participants []north.User `json:"participants"`
	LastMessage  *DMMessage   `json:"lastMessage"`
	UpdatedAt    time.Time    `json:"updatedAt"`
	UnreadCount  int          `json:"unreadCount"`
}

// DMConversationPage is one cursor-paginated conversation page.
type DMConversationPage struct {
	Items        []DMConversation `json:"items"`
	NextCursor   *string          `json:"nextCursor"`
	RequestCount int              `json:"requestCount"`
}

// DMMessage is one direct message.
type DMMessage struct {
	ID        string       `json:"id"`
	Text      string       `json:"text"`
	Sender    north.User   `json:"sender"`
	Media     []Media      `json:"media"`
	Post      *Post        `json:"tweet,omitempty"`
	CreatedAt time.Time    `json:"createdAt"`
	EditedAt  *time.Time   `json:"editedAt"`
	System    bool         `json:"system"`
	Read      bool         `json:"read"`
	Reactions []DMReaction `json:"reactions"`
	ReplyTo   *DMReply     `json:"replyTo"`
}

// DMReaction groups one emoji and the accounts that used it.
type DMReaction struct {
	Emoji           string       `json:"emoji"`
	Users           []north.User `json:"users"`
	Count           int          `json:"count"`
	ReactedByViewer bool         `json:"reactedByViewer"`
}

// DMReply is the compact message embedded as a reply target.
type DMReply struct {
	ID       string      `json:"id"`
	Text     string      `json:"text"`
	Sender   *north.User `json:"sender"`
	HasMedia bool        `json:"hasMedia"`
	Deleted  bool        `json:"deleted"`
}

// DMMessagePage contains messages and their conversation metadata.
type DMMessagePage struct {
	Items        []DMMessage    `json:"items"`
	NextCursor   *string        `json:"nextCursor"`
	Conversation DMConversation `json:"conversation"`
}

// SendDMRequest is the content of a new direct message.
type SendDMRequest struct {
	Text      string   `json:"text,omitempty"`
	MediaIDs  []string `json:"mediaIds,omitempty"`
	ReplyToID string   `json:"replyToId,omitempty"`
}

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

// Trend is one entry in north's current trend list.
type Trend struct {
	Tag       string `json:"tag"`
	Count     int    `json:"count"`
	IsHashtag bool   `json:"isHashtag"`
}

// TrendList contains the trends returned by the web application.
type TrendList struct {
	Items []Trend `json:"items"`
}
