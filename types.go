package north

import "time"

// ReplyPolicy is the audience allowed to reply to a post.
type ReplyPolicy string

const (
	ReplyEveryone  ReplyPolicy = "EVERYONE"
	ReplyFollowing ReplyPolicy = "FOLLOWING"
	ReplyMentioned ReplyPolicy = "MENTIONED"
)

// HiddenReason is why a post would normally be hidden.
type HiddenReason string

const (
	HiddenMuted   HiddenReason = "muted"
	HiddenBlocked HiddenReason = "blocked"
)

// MediaKind is the kind of media attached to a post.
type MediaKind string

const (
	MediaPhoto MediaKind = "PHOTO"
	MediaGIF   MediaKind = "GIF"
	MediaVideo MediaKind = "VIDEO"
)

// MediaStatus is the server-side processing state of uploaded media.
type MediaStatus string

const (
	MediaPending MediaStatus = "PENDING"
	MediaReady   MediaStatus = "READY"
	MediaFailed  MediaStatus = "FAILED"
)

// MediaWarning is the sensitive-content warning on an attachment.
type MediaWarning string

const (
	WarningNudity    MediaWarning = "NUDITY"
	WarningViolence  MediaWarning = "VIOLENCE"
	WarningSensitive MediaWarning = "SENSITIVE"
)

// SearchTab is the order used for search results.
type SearchTab string

const (
	SearchLatest SearchTab = "latest"
	SearchTop    SearchTab = "top"
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

// Post is a north post. Quoted and RepostOf are expanded by at most one level
// by the service.
type Post struct {
	ID                string       `json:"id"`
	Text              string       `json:"text"`
	CreatedAt         time.Time    `json:"createdAt"`
	EditedAt          *time.Time   `json:"editedAt"`
	EditCount         int          `json:"editCount"`
	Author            User         `json:"author"`
	ConversationID    string       `json:"conversationId"`
	ReplyPolicy       ReplyPolicy  `json:"replyPolicy"`
	Source            string       `json:"source"`
	InReplyToID       *string      `json:"inReplyToId"`
	InReplyToHandle   *string      `json:"inReplyToHandle"`
	Quoted            *Post        `json:"quoted"`
	RepostOf          *Post        `json:"retweetOf"`
	LikeCount         int          `json:"likeCount"`
	RepostCount       int          `json:"retweetCount"`
	ReplyCount        int          `json:"replyCount"`
	QuoteCount        int          `json:"quoteCount"`
	Media             []Media      `json:"media"`
	Poll              *Poll        `json:"poll"`
	Liked             bool         `json:"liked"`
	Reposted          bool         `json:"retweeted"`
	Bookmarked        bool         `json:"bookmarked"`
	Deleted           bool         `json:"deleted"`
	Unavailable       bool         `json:"unavailable"`
	QuotedUnavailable bool         `json:"quotedUnavailable"`
	HiddenReason      HiddenReason `json:"hiddenReason"`
}

// DisplayPost returns the original post for a repost, and the receiver for a
// normal post. It is useful for clients that render repost attribution around
// the original content.
func (p *Post) DisplayPost() *Post {
	if p != nil && p.RepostOf != nil {
		return p.RepostOf
	}

	return p
}

// Media is an uploaded attachment.
type Media struct {
	ID           string        `json:"id"`
	Kind         MediaKind     `json:"kind"`
	Status       MediaStatus   `json:"status"`
	URL          string        `json:"url"`
	ThumbnailURL *string       `json:"thumbnailUrl"`
	Width        int           `json:"width"`
	Height       int           `json:"height"`
	DurationMS   *int64        `json:"durationMs"`
	AltText      *string       `json:"altText"`
	Sensitive    bool          `json:"sensitive"`
	Warning      *MediaWarning `json:"warning"`
}

// Poll is a poll attached to a post.
type Poll struct {
	EndsAt  time.Time    `json:"endsAt"`
	Voted   *int         `json:"voted"`
	Options []PollOption `json:"options"`
}

// PollOption is one answer in a poll.
type PollOption struct {
	Label string `json:"label"`
	Votes int    `json:"votes"`
}

// PostPage is one cursor-paginated page of posts.
type PostPage struct {
	Items      []Post  `json:"items"`
	NextCursor *string `json:"nextCursor"`
}

// UserPage is one cursor-paginated page of accounts.
type UserPage struct {
	Items      []User  `json:"items"`
	NextCursor *string `json:"nextCursor"`
}

// CreatePostRequest is the request body for a new post. Text and Media are
// individually optional, but the API requires at least one of them.
type CreatePostRequest struct {
	Text        string           `json:"text,omitempty"`
	Media       *CreatePostMedia `json:"media,omitempty"`
	Reply       *CreatePostReply `json:"reply,omitempty"`
	QuotePostID string           `json:"quote_tweet_id,omitempty"`
}

// CreatePostMedia contains IDs returned by a media upload. The API accepts at
// most four IDs.
type CreatePostMedia struct {
	MediaIDs []string `json:"media_ids"`
}

// CreatePostReply identifies the post being replied to.
type CreatePostReply struct {
	InReplyToPostID string `json:"in_reply_to_tweet_id"`
}

// CreatedPost is the compact response returned after creating a post.
type CreatedPost struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// SearchOptions controls a post search.
type SearchOptions struct {
	Tab    SearchTab
	Cursor string
}

// TimelineOptions controls the home timeline.
type TimelineOptions struct {
	Ranked bool
	Cursor string
}

// LikeState is the state after liking or unliking a post.
type LikeState struct {
	Liked     bool `json:"liked"`
	LikeCount int  `json:"likeCount"`
}

// RepostState is the state after reposting or undoing a repost.
type RepostState struct {
	Reposted    bool `json:"retweeted"`
	RepostCount int  `json:"retweetCount"`
}

// UploadSession starts a chunked media upload.
type UploadSession struct {
	UploadID  string `json:"uploadId"`
	ChunkSize int64  `json:"chunkSize"`
}

// UploadProgress reports the cumulative bytes accepted for a chunked upload.
type UploadProgress struct {
	Received int64 `json:"received"`
}
