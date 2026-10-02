package oauth

import "golang.org/x/oauth2"

const RevokeURL = "https://api.north.rip/oauth/revoke"

// Endpoint contains north's public OAuth endpoints.
var Endpoint = oauth2.Endpoint{
	AuthURL:       "https://api.north.rip/oauth/authorize",
	DeviceAuthURL: "https://api.north.rip/oauth/device/code",
	TokenURL:      "https://api.north.rip/oauth/token",
	AuthStyle:     oauth2.AuthStyleInParams,
}

// NewConfig returns a standard OAuth configuration for north.
func NewConfig(clientID, redirectURL string, scopes ...string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:    clientID,
		RedirectURL: redirectURL,
		Scopes:      scopes,
		Endpoint:    Endpoint,
	}
}
