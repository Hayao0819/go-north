package oauth

import (
	"net/url"
	"testing"

	"golang.org/x/oauth2"
)

func TestAuthorizationURL(t *testing.T) {
	t.Parallel()

	config := NewConfig("client", "http://127.0.0.1/callback", "posts.read")
	config.Endpoint.AuthURL = "https://api.example.test/oauth/authorize"
	verifier := oauth2.GenerateVerifier()
	authorizationURL := config.AuthCodeURL("state", oauth2.S256ChallengeOption(verifier))
	parsed, err := url.Parse(authorizationURL)
	if err != nil {
		t.Fatal(err)
	}
	query := parsed.Query()
	if parsed.Path != "/oauth/authorize" ||
		query.Get("client_id") != "client" ||
		query.Get("redirect_uri") != "http://127.0.0.1/callback" ||
		query.Get("scope") != "posts.read" ||
		query.Get("code_challenge") != oauth2.S256ChallengeFromVerifier(verifier) ||
		query.Get("code_challenge_method") != "S256" ||
		query.Get("state") != "state" {
		t.Errorf("AuthorizationURL = %s", authorizationURL)
	}
}
