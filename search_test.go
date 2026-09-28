package north

import (
	"testing"
	"time"
)

func TestSearchQuery(t *testing.T) {
	t.Parallel()

	query := SearchQuery{
		AllWords:    []string{"north bugs"},
		ExactPhrase: "thank you",
		AnyWords:    []string{"images", "videos"},
		Without:     []string{"spam ads"},
		Hashtags:    []string{"go", "#sdk"},
		From:        " @Alice ",
		To:          "@BOB",
		Mentions:    []string{"@Carol dave"},
		Replies:     SearchFilterOnly,
		Links:       SearchFilterExclude,
		MinReplies:  2,
		MinLikes:    3,
		MinReposts:  4,
		Since:       time.Date(2026, time.September, 1, 12, 0, 0, 0, time.FixedZone("test", 9*60*60)),
		Until:       time.Date(2026, time.September, 30, 0, 0, 0, 0, time.UTC),
	}

	want := `north bugs "thank you" (images OR videos) -spam -ads #go #sdk from:alice to:bob @carol @dave filter:replies -filter:links min_replies:2 min_faves:3 min_retweets:4 since:2026-09-01 until:2026-09-30`
	if got := query.String(); got != want {
		t.Errorf("SearchQuery.String() = %q, want %q", got, want)
	}
}

func TestSearchQueryOmitsEmptyFilters(t *testing.T) {
	t.Parallel()

	query := SearchQuery{
		AnyWords:   []string{"north"},
		MinReplies: -1,
		MinLikes:   0,
		Replies:    SearchFilterInclude,
	}

	if got := query.String(); got != "north" {
		t.Errorf("SearchQuery.String() = %q, want %q", got, "north")
	}
}
