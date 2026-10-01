package north

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Hayao0819/go-north/internal/transport"
)

const (
	defaultNotificationRetry = 5 * time.Second
	maxNotificationEventSize = 1 << 20
)

// NotificationTab selects a notification feed.
type NotificationTab string

const (
	// NotificationsAll includes every notification visible to the account.
	NotificationsAll NotificationTab = "all"
	// NotificationsVerified includes notifications from verified accounts.
	NotificationsVerified NotificationTab = "verified"
	// NotificationsMentions includes replies and mentions of the account.
	NotificationsMentions NotificationTab = "mentions"
)

// NotificationKind identifies why a notification was created.
type NotificationKind string

const (
	NotificationFollow  NotificationKind = "FOLLOW"
	NotificationLike    NotificationKind = "LIKE"
	NotificationRepost  NotificationKind = "RETWEET"
	NotificationQuote   NotificationKind = "QUOTE"
	NotificationReply   NotificationKind = "REPLY"
	NotificationMention NotificationKind = "MENTION"
	NotificationPost    NotificationKind = "POST"
)

// Notification is one grouped item in the notification feed.
type Notification struct {
	ID          string           `json:"id"`
	Kind        NotificationKind `json:"kind"`
	Actors      []User           `json:"actors"`
	ActorCount  int              `json:"actorCount"`
	GroupCount  int              `json:"groupCount"`
	TargetCount int              `json:"targetCount"`
	Post        *Post            `json:"tweet"`
	CreatedAt   time.Time        `json:"createdAt"`
	Read        bool             `json:"read"`
}

// NotificationPage is one cursor-paginated notification page.
type NotificationPage struct {
	Items      []Notification `json:"items"`
	NextCursor *string        `json:"nextCursor"`
}

// NotificationEvent announces a new notification without its full content.
type NotificationEvent struct {
	ID        string           `json:"id"`
	Kind      NotificationKind `json:"kind"`
	PostID    string           `json:"tweetId"`
	CreatedAt time.Time        `json:"createdAt"`
}

// NotificationStream reads notification events from a server-sent event
// connection. It is not safe to call Receive concurrently.
type NotificationStream struct {
	scanner    *bufio.Scanner
	body       io.ReadCloser
	retryAfter time.Duration
	closeOnce  sync.Once
	closeErr   error
}

// Notifications returns one page from the selected notification tab.
func (c *Client) Notifications(ctx context.Context, tab NotificationTab, cursor string) (NotificationPage, *Response, error) {
	if tab == "" {
		tab = NotificationsAll
	}
	query := url.Values{"tab": {string(tab)}}
	setCursor(query, cursor)

	return doData[NotificationPage](ctx, c, http.MethodGet, "/2/notifications", query, nil, "")
}

// NotificationUnreadCount returns the current unread notification count.
func (c *Client) NotificationUnreadCount(ctx context.Context) (int, *Response, error) {
	data, response, err := doData[struct {
		Count int `json:"count"`
	}](ctx, c, http.MethodGet, "/2/notifications/unread_count", nil, nil, "")

	return data.Count, response, err
}

// MarkNotificationsRead marks visible notifications as read and returns the
// number changed by the service.
func (c *Client) MarkNotificationsRead(ctx context.Context) (int, *Response, error) {
	data, response, err := doData[struct {
		Marked int `json:"marked"`
	}](ctx, c, http.MethodPost, "/2/notifications/read", nil, nil, "")

	return data.Marked, response, err
}

// StreamNotifications opens the server-sent event notification stream. The
// caller must close the returned stream.
func (c *Client) StreamNotifications(ctx context.Context) (*NotificationStream, *Response, error) {
	raw, err := c.streamingTransport.Open(
		ctx,
		http.MethodGet,
		"/2/notifications/stream",
		nil,
		nil,
		"",
		"text/event-stream",
	)
	if err != nil {
		return nil, nil, fmt.Errorf("north: %w", err)
	}

	response := newResponse(transport.Result{
		StatusCode: raw.StatusCode,
		Status:     raw.Status,
		Header:     raw.Header.Clone(),
	})
	if raw.StatusCode < http.StatusOK || raw.StatusCode >= http.StatusMultipleChoices {
		defer raw.Body.Close()
		result, readErr := transport.ReadResponse(raw)
		if readErr != nil {
			if errors.Is(readErr, transport.ErrResponseTooLarge) {
				return nil, response, ErrResponseTooLarge
			}

			return nil, response, fmt.Errorf("north: %w", readErr)
		}

		return nil, response, decodeAPIError(result, response)
	}

	mediaType, _, mediaErr := mime.ParseMediaType(raw.Header.Get("Content-Type"))
	if mediaErr != nil || mediaType != "text/event-stream" {
		_ = raw.Body.Close()

		return nil, response, fmt.Errorf("north: notification stream returned content type %q", raw.Header.Get("Content-Type"))
	}

	scanner := bufio.NewScanner(raw.Body)
	scanner.Buffer(make([]byte, 4096), maxNotificationEventSize)

	return &NotificationStream{
		scanner:    scanner,
		body:       raw.Body,
		retryAfter: defaultNotificationRetry,
	}, response, nil
}

// Receive waits for and returns the next notification event. Heartbeats and
// unrelated event types are skipped.
func (s *NotificationStream) Receive() (NotificationEvent, error) {
	var (
		eventName string
		data      strings.Builder
	)
	for s.scanner.Scan() {
		line := strings.TrimSuffix(s.scanner.Text(), "\r")
		if line == "" {
			if data.Len() == 0 {
				eventName = ""
				continue
			}
			if eventName != "" && eventName != "notification" {
				eventName = ""
				data.Reset()
				continue
			}

			var event NotificationEvent
			payload := strings.TrimSuffix(data.String(), "\n")
			if err := json.Unmarshal([]byte(payload), &event); err != nil {
				return NotificationEvent{}, fmt.Errorf("north: decode notification event: %w", err)
			}

			return event, nil
		}
		if strings.HasPrefix(line, ":") {
			continue
		}

		field, value, _ := strings.Cut(line, ":")
		value = strings.TrimPrefix(value, " ")
		switch field {
		case "event":
			eventName = value
		case "data":
			data.WriteString(value)
			data.WriteByte('\n')
		case "retry":
			milliseconds, err := strconv.ParseInt(value, 10, 64)
			if err == nil && milliseconds >= 0 && milliseconds <= int64((time.Duration(1<<63-1))/time.Millisecond) {
				s.retryAfter = time.Duration(milliseconds) * time.Millisecond
			}
		}
	}
	if err := s.scanner.Err(); err != nil {
		return NotificationEvent{}, fmt.Errorf("north: read notification stream: %w", err)
	}

	return NotificationEvent{}, io.EOF
}

// RetryAfter returns the server's requested delay before reconnecting.
func (s *NotificationStream) RetryAfter() time.Duration {
	return s.retryAfter
}

// Close closes the event stream.
func (s *NotificationStream) Close() error {
	if s == nil || s.body == nil {
		return nil
	}
	s.closeOnce.Do(func() {
		s.closeErr = s.body.Close()
	})

	return s.closeErr
}
