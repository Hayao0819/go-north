package unofficial

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLoginCreatesSession(t *testing.T) {
	t.Parallel()

	startedAt := time.Unix(1_800_000_000, 123_000_000)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/auth/turnstile":
			writeJSON(t, writer, http.StatusOK, `{"enabled":true,"siteKey":"site-key"}`)
		case "/api/auth/login":
			var body struct {
				Identifier     string `json:"identifier"`
				Password       string `json:"password"`
				Nickname       string `json:"nickname"`
				StartedAt      int64  `json:"startedAt"`
				Automated      bool   `json:"automated"`
				TurnstileToken string `json:"turnstileToken"`
			}
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode login: %v", err)
			}
			if body.Identifier != "alice" || body.Password != "secret" || body.Nickname != "" || body.Automated {
				t.Errorf("login body = %#v", body)
			}
			if body.StartedAt != startedAt.UnixMilli() || body.TurnstileToken != "turnstile" {
				t.Errorf("login verification = %#v", body)
			}
			http.SetCookie(writer, &http.Cookie{Name: "session", Value: "logged-in", Path: "/api", HttpOnly: true})
			writeJSON(t, writer, http.StatusOK, `{}`)
		case "/api/auth/me":
			cookie, err := request.Cookie("session")
			if err != nil || cookie.Value != "logged-in" {
				t.Errorf("session cookie = %#v, %v", cookie, err)
			}
			writeJSON(t, writer, http.StatusOK, `{"user":{"id":"1","handle":"alice","name":"Alice"}}`)
		default:
			http.NotFound(writer, request)
		}
	}))
	t.Cleanup(server.Close)

	options := []Option{WithBaseURL(server.URL), WithHTTPClient(server.Client())}
	requirements, _, err := GetLoginRequirements(context.Background(), options...)
	if err != nil {
		t.Fatal(err)
	}
	if !requirements.TurnstileEnabled || requirements.TurnstileSiteKey != "site-key" {
		t.Fatalf("requirements = %#v", requirements)
	}

	client, result, response, err := Login(context.Background(), LoginRequest{
		Identifier:     " alice ",
		Password:       "secret",
		TurnstileToken: "turnstile",
		StartedAt:      startedAt,
	}, options...)
	if err != nil {
		t.Fatal(err)
	}
	if result.RequiresTwoFactor || response.StatusCode != http.StatusOK {
		t.Fatalf("login = %#v, %#v", result, response)
	}
	if got := client.CookieHeader(); got != "session=logged-in" {
		t.Fatalf("cookie header = %q", got)
	}
	user, _, err := client.Me(context.Background())
	if err != nil || user.ID != "1" || user.Handle != "alice" {
		t.Fatalf("Me = %#v, %v", user, err)
	}
}

func TestLoginCompletesTwoFactor(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/auth/login":
			writeJSON(t, writer, http.StatusOK, `{"requires2fa":true,"token":"pending"}`)
		case "/api/auth/login/2fa":
			var body struct {
				Token string `json:"token"`
				Code  string `json:"code"`
			}
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode two-factor request: %v", err)
			}
			if body.Token != "pending" || body.Code != "123456" {
				t.Errorf("two-factor body = %#v", body)
			}
			http.SetCookie(writer, &http.Cookie{Name: "session", Value: "verified", Path: "/"})
			writer.WriteHeader(http.StatusNoContent)
		case "/api/auth/me":
			writeJSON(t, writer, http.StatusOK, `{"user":{"id":"1","handle":"alice"}}`)
		default:
			http.NotFound(writer, request)
		}
	}))
	t.Cleanup(server.Close)

	client, result, _, err := Login(context.Background(), LoginRequest{
		Identifier: "alice",
		Password:   "secret",
	}, WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	if !result.RequiresTwoFactor || result.TwoFactorToken != "pending" {
		t.Fatalf("login result = %#v", result)
	}
	if _, err := client.CompleteTwoFactor(context.Background(), result.TwoFactorToken, "123456"); err != nil {
		t.Fatal(err)
	}
	if got := client.CookieHeader(); got != "session=verified" {
		t.Fatalf("cookie header = %q", got)
	}
}

func TestLoginValidatesCredentialsAndSession(t *testing.T) {
	t.Parallel()

	if _, _, _, err := Login(context.Background(), LoginRequest{Password: "secret"}); !errors.Is(err, ErrLoginIdentifierRequired) {
		t.Fatalf("identifier error = %v", err)
	}
	if _, _, _, err := Login(context.Background(), LoginRequest{Identifier: "alice"}); !errors.Is(err, ErrLoginPasswordRequired) {
		t.Fatalf("password error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(t, writer, http.StatusOK, `{}`)
	}))
	t.Cleanup(server.Close)
	_, _, _, err := Login(context.Background(), LoginRequest{
		Identifier: "alice",
		Password:   "secret",
	}, WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	if !errors.Is(err, ErrSessionNotEstablished) {
		t.Fatalf("missing session error = %v", err)
	}
}

func TestMeRejectsAnEmptySession(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(server.Close)
	client, err := NewClient("session=expired", WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = client.Me(context.Background())
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("Me error = %v", err)
	}
}

func TestLogoutEndpoint(t *testing.T) {
	t.Parallel()

	testEndpoints(t, []endpointTest{
		{"logout", http.MethodPost, "/api/auth/logout", nil, nil, func(ctx context.Context, client *Client) error {
			_, err := client.Logout(ctx)
			return err
		}},
	})
}
