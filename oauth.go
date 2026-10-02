package north

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	northoauth "github.com/Hayao0819/go-north/oauth"
	"golang.org/x/oauth2"
)

// Scope is a permission granted to an OAuth application or personal token.
type Scope string

const (
	ScopePostsRead            Scope = "posts.read"
	ScopeUsersRead            Scope = "users.read"
	ScopePostsWrite           Scope = "posts.write"
	ScopePostsEdit            Scope = "posts.edit"
	ScopePostsDelete          Scope = "posts.delete"
	ScopePollsVote            Scope = "polls.vote"
	ScopeReactionsWrite       Scope = "reactions.write"
	ScopeFollowsWrite         Scope = "follows.write"
	ScopeModerationRead       Scope = "moderation.read"
	ScopeModerationWrite      Scope = "moderation.write"
	ScopeBookmarksRead        Scope = "bookmarks.read"
	ScopeBookmarksWrite       Scope = "bookmarks.write"
	ScopeProfileWrite         Scope = "profile.write"
	ScopeNotificationsRead    Scope = "notifications.read"
	ScopeNotificationsWrite   Scope = "notifications.write"
	ScopeTrendsRead           Scope = "trends.read"
	ScopeListsRead            Scope = "lists.read"
	ScopeListsWrite           Scope = "lists.write"
	ScopeTrendsWrite          Scope = "trends.write"
	ScopeMediaWrite           Scope = "media.write"
	ScopeDraftsRead           Scope = "drafts.read"
	ScopeDraftsWrite          Scope = "drafts.write"
	ScopePostsSchedule        Scope = "posts.schedule"
	ScopeDMRead               Scope = "dm.read"
	ScopeDMWrite              Scope = "dm.write"
	ScopeDMReceiptsWrite      Scope = "dm.receipts.write"
	ScopeDMDelete             Scope = "dm.delete"
	ScopeDMConversationsWrite Scope = "dm.conversations.write"
	ScopeDMRequestsWrite      Scope = "dm.requests.write"
)

// ErrOAuthConfigRequired is returned when an OAuth client is created without
// its configuration.
var ErrOAuthConfigRequired = errors.New("north: OAuth config is required")

// SaveTokenFunc persists an OAuth token. It must atomically replace the
// previous token and must not modify token.
type SaveTokenFunc func(token *oauth2.Token) error

// NewOAuthConfig returns north's OAuth configuration.
func NewOAuthConfig(clientID, redirectURL string, scopes ...Scope) *oauth2.Config {
	values := make([]string, len(scopes))
	for index, scope := range scopes {
		values[index] = string(scope)
	}

	return northoauth.NewConfig(clientID, redirectURL, values...)
}

// NewClientWithOAuth returns a Client backed by an automatically refreshed
// OAuth token. save is called before a refreshed token is used; it may be nil.
func NewClientWithOAuth(
	ctx context.Context,
	config *oauth2.Config,
	token *oauth2.Token,
	save SaveTokenFunc,
	opts ...Option,
) (*Client, error) {
	if config == nil {
		return nil, ErrOAuthConfigRequired
	}
	if token == nil || (strings.TrimSpace(token.AccessToken) == "" && strings.TrimSpace(token.RefreshToken) == "") {
		return nil, ErrTokenRequired
	}

	source := config.TokenSource(ctx, token)
	if save != nil {
		source = &savingTokenSource{
			source: source,
			state:  tokenStateOf(token),
			save:   save,
		}
	}

	return NewClientWithTokenSource(source, opts...)
}

type savingTokenSource struct {
	source oauth2.TokenSource
	state  tokenState
	save   SaveTokenFunc
	mu     sync.Mutex
}

func (s *savingTokenSource) Token() (*oauth2.Token, error) {
	token, err := s.source.Token()
	if err != nil || token == nil {
		return token, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	state := tokenStateOf(token)
	if state == s.state {
		return token, nil
	}
	if err := s.save(token); err != nil {
		return nil, err
	}
	s.state = state

	return token, nil
}

type tokenState struct {
	accessToken  string
	tokenType    string
	refreshToken string
	expiry       time.Time
}

func tokenStateOf(token *oauth2.Token) tokenState {
	if token == nil {
		return tokenState{}
	}

	return tokenState{
		accessToken:  token.AccessToken,
		tokenType:    token.TokenType,
		refreshToken: token.RefreshToken,
		expiry:       token.Expiry,
	}
}
