package north

import "strings"

// TokenKind identifies the token format documented by north. It does not
// validate the token or its scopes.
type TokenKind uint8

const (
	TokenUnknown TokenKind = iota
	TokenLegacy
	TokenScoped
)

// DetectTokenKind identifies a token by its documented prefix. OAuth access
// tokens and personal tokens are both scoped nth_oat_ tokens.
func DetectTokenKind(token string) TokenKind {
	token = strings.TrimSpace(token)
	switch {
	case strings.HasPrefix(token, "nth_live_"):
		return TokenLegacy
	case strings.HasPrefix(token, "nth_oat_"):
		return TokenScoped
	default:
		return TokenUnknown
	}
}

// TokenKind returns the format of the token supplied to NewClient.
func (c *Client) TokenKind() TokenKind {
	return c.tokenKind
}
