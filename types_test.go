package north

import "testing"

func TestDisplayPost(t *testing.T) {
	t.Parallel()

	original := &Post{ID: "original"}
	repost := &Post{ID: "repost", RepostOf: original}

	if got := original.DisplayPost(); got != original {
		t.Errorf("normal DisplayPost = %p, want %p", got, original)
	}
	if got := repost.DisplayPost(); got != original {
		t.Errorf("repost DisplayPost = %p, want %p", got, original)
	}
	var missing *Post
	if got := missing.DisplayPost(); got != nil {
		t.Errorf("nil DisplayPost = %#v", got)
	}
}
