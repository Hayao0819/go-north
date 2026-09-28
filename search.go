package north

import (
	"strconv"
	"strings"
	"time"
)

// SearchFilter controls whether replies or links are required or excluded.
type SearchFilter uint8

const (
	SearchFilterInclude SearchFilter = iota
	SearchFilterOnly
	SearchFilterExclude
)

// SearchQuery describes the filters accepted by north's post search.
type SearchQuery struct {
	AllWords    []string
	ExactPhrase string
	AnyWords    []string
	Without     []string
	Hashtags    []string

	From     string
	To       string
	Mentions []string

	Replies SearchFilter
	Links   SearchFilter

	MinReplies int
	MinLikes   int
	MinReposts int

	Since time.Time
	Until time.Time
}

// String returns the query in north's search syntax.
func (q SearchQuery) String() string {
	parts := fields(q.AllWords)

	if phrase := strings.TrimSpace(q.ExactPhrase); phrase != "" {
		parts = append(parts, `"`+phrase+`"`)
	}

	any := fields(q.AnyWords)
	if len(any) == 1 {
		parts = append(parts, any[0])
	} else if len(any) > 1 {
		parts = append(parts, "("+strings.Join(any, " OR ")+")")
	}

	for _, word := range fields(q.Without) {
		parts = append(parts, "-"+word)
	}
	for _, tag := range fields(q.Hashtags) {
		if !strings.HasPrefix(tag, "#") {
			tag = "#" + tag
		}
		parts = append(parts, tag)
	}

	if handle := searchHandle(q.From); handle != "" {
		parts = append(parts, "from:"+handle)
	}
	if handle := searchHandle(q.To); handle != "" {
		parts = append(parts, "to:"+handle)
	}
	for _, handle := range fields(q.Mentions) {
		if handle = searchHandle(handle); handle != "" {
			parts = append(parts, "@"+handle)
		}
	}

	parts = appendSearchFilter(parts, q.Replies, "replies")
	parts = appendSearchFilter(parts, q.Links, "links")

	if q.MinReplies > 0 {
		parts = append(parts, "min_replies:"+strconv.Itoa(q.MinReplies))
	}
	if q.MinLikes > 0 {
		parts = append(parts, "min_faves:"+strconv.Itoa(q.MinLikes))
	}
	if q.MinReposts > 0 {
		parts = append(parts, "min_retweets:"+strconv.Itoa(q.MinReposts))
	}

	if !q.Since.IsZero() {
		parts = append(parts, "since:"+q.Since.Format(time.DateOnly))
	}
	if !q.Until.IsZero() {
		parts = append(parts, "until:"+q.Until.Format(time.DateOnly))
	}

	return strings.Join(parts, " ")
}

func fields(values []string) []string {
	var result []string
	for _, value := range values {
		result = append(result, strings.Fields(value)...)
	}

	return result
}

func searchHandle(value string) string {
	return strings.ToLower(strings.TrimPrefix(strings.TrimSpace(value), "@"))
}

func appendSearchFilter(parts []string, filter SearchFilter, name string) []string {
	switch filter {
	case SearchFilterOnly:
		return append(parts, "filter:"+name)
	case SearchFilterExclude:
		return append(parts, "-filter:"+name)
	default:
		return parts
	}
}
