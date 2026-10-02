package north

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func TestNewOAuthConfig(t *testing.T) {
	t.Parallel()

	config := NewOAuthConfig("client", "http://127.0.0.1/callback", ScopePostsRead, ScopeUsersRead)
	if config.ClientID != "client" || config.ClientSecret != "" {
		t.Fatalf("client credentials = %q, %q", config.ClientID, config.ClientSecret)
	}
	if config.RedirectURL != "http://127.0.0.1/callback" {
		t.Fatalf("RedirectURL = %q", config.RedirectURL)
	}
	if len(config.Scopes) != 2 || config.Scopes[0] != "posts.read" || config.Scopes[1] != "users.read" {
		t.Fatalf("Scopes = %#v", config.Scopes)
	}
	if config.Endpoint.AuthStyle != oauth2.AuthStyleInParams {
		t.Fatalf("AuthStyle = %v", config.Endpoint.AuthStyle)
	}
}

func TestNewClientWithOAuthRefresh(t *testing.T) {
	t.Parallel()

	var tokenRequests atomic.Int32
	var apiRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/oauth/token":
			tokenRequests.Add(1)
			if err := request.ParseForm(); err != nil {
				t.Errorf("ParseForm: %v", err)
			}
			if request.Header.Get("Authorization") != "" || request.Form.Has("client_secret") {
				t.Errorf("client secret was sent: header=%q form=%#v", request.Header.Get("Authorization"), request.Form)
			}
			if request.Form.Get("client_id") != "client" || request.Form.Get("grant_type") != "refresh_token" || request.Form.Get("refresh_token") != "old-refresh" {
				t.Errorf("refresh form = %#v", request.Form)
			}
			writeJSON(t, writer, http.StatusOK, `{"access_token":"nth_oat_new","refresh_token":"new-refresh","token_type":"Bearer","expires_in":3600,"scope":"users.read"}`)
		case "/2/users/me":
			apiRequests.Add(1)
			if request.Header.Get("Authorization") != "Bearer nth_oat_new" {
				t.Errorf("Authorization = %q", request.Header.Get("Authorization"))
			}
			writeJSON(t, writer, http.StatusOK, `{"data":{"id":"1","handle":"north","name":"North"}}`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	config := NewOAuthConfig("client", "", ScopeUsersRead)
	config.Endpoint.TokenURL = server.URL + "/oauth/token"
	initial := &oauth2.Token{
		AccessToken:  "nth_oat_old",
		RefreshToken: "old-refresh",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(-time.Hour),
	}
	var saved *oauth2.Token
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, server.Client())
	client, err := NewClientWithOAuth(
		ctx,
		config,
		initial,
		func(token *oauth2.Token) error {
			saved = token
			return nil
		},
		WithBaseURL(server.URL),
		WithHTTPClient(server.Client()),
	)
	if err != nil {
		t.Fatal(err)
	}

	user, _, err := client.Me(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if user.ID != "1" || client.TokenKind() != TokenScoped {
		t.Fatalf("user = %#v, token kind = %v", user, client.TokenKind())
	}
	if saved == nil || saved.AccessToken != "nth_oat_new" || saved.RefreshToken != "new-refresh" {
		t.Fatalf("saved token = %#v", saved)
	}
	if tokenRequests.Load() != 1 || apiRequests.Load() != 1 {
		t.Fatalf("requests = token:%d API:%d", tokenRequests.Load(), apiRequests.Load())
	}
}

func TestNewClientWithOAuthValidation(t *testing.T) {
	t.Parallel()

	token := &oauth2.Token{AccessToken: "nth_oat_token"}
	if _, err := NewClientWithOAuth(context.Background(), nil, token, nil); !errors.Is(err, ErrOAuthConfigRequired) {
		t.Fatalf("nil config error = %v", err)
	}
	if _, err := NewClientWithOAuth(context.Background(), NewOAuthConfig("client", ""), nil, nil); !errors.Is(err, ErrTokenRequired) {
		t.Fatalf("nil token error = %v", err)
	}
	if _, err := NewClientWithOAuth(context.Background(), NewOAuthConfig("client", ""), &oauth2.Token{}, nil); !errors.Is(err, ErrTokenRequired) {
		t.Fatalf("empty token error = %v", err)
	}
}

func TestSavingTokenSourceRetriesSave(t *testing.T) {
	t.Parallel()

	rotated := &oauth2.Token{AccessToken: "new", RefreshToken: "rotated"}
	var calls atomic.Int32
	source := &savingTokenSource{
		source: tokenSourceFunc(func() (*oauth2.Token, error) {
			return rotated, nil
		}),
		save: func(*oauth2.Token) error {
			if calls.Add(1) == 1 {
				return errors.New("keyring unavailable")
			}
			return nil
		},
	}

	if _, err := source.Token(); err == nil {
		t.Fatal("Token succeeded when saving failed")
	}
	if _, err := source.Token(); err != nil {
		t.Fatalf("Token retry: %v", err)
	}
	if _, err := source.Token(); err != nil {
		t.Fatalf("Token reuse: %v", err)
	}
	if calls.Load() != 2 {
		t.Fatalf("save calls = %d", calls.Load())
	}
}

type tokenSourceFunc func() (*oauth2.Token, error)

func (f tokenSourceFunc) Token() (*oauth2.Token, error) {
	return f()
}
