package unofficial

import (
	"context"
	"errors"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

var (
	// ErrLoginIdentifierRequired is returned when a login has no email address or handle.
	ErrLoginIdentifierRequired = errors.New("north unofficial: login identifier is required")
	// ErrLoginPasswordRequired is returned when a login has no password.
	ErrLoginPasswordRequired = errors.New("north unofficial: login password is required")
	// ErrTwoFactorTokenRequired is returned when the first login step did not supply a token.
	ErrTwoFactorTokenRequired = errors.New("north unofficial: two-factor token is required")
	// ErrTwoFactorCodeRequired is returned when no authenticator or backup code was supplied.
	ErrTwoFactorCodeRequired = errors.New("north unofficial: two-factor code is required")
	// ErrSessionNotEstablished is returned when login succeeded without setting a session cookie.
	ErrSessionNotEstablished = errors.New("north unofficial: login did not establish a session")
	// ErrNotAuthenticated is returned when the current session has no account.
	ErrNotAuthenticated = errors.New("north unofficial: session is not authenticated")
)

// GetLoginRequirements returns the current login verification settings.
func GetLoginRequirements(ctx context.Context, opts ...Option) (LoginRequirements, *Response, error) {
	client, err := newLoginClient(opts...)
	if err != nil {
		return LoginRequirements{}, nil, err
	}

	return doJSON[LoginRequirements](ctx, client, http.MethodGet, "/api/auth/turnstile", nil, nil)
}

// Login attempts to start a web session with an email address or handle and
// password. It is experimental: callers must obtain any required Turnstile
// token themselves, and north may reject non-browser login attempts. When
// RequiresTwoFactor is true, call CompleteTwoFactor on the returned client.
func Login(ctx context.Context, request LoginRequest, opts ...Option) (*Client, LoginResult, *Response, error) {
	request.Identifier = strings.TrimSpace(request.Identifier)
	if request.Identifier == "" {
		return nil, LoginResult{}, nil, ErrLoginIdentifierRequired
	}
	if request.Password == "" {
		return nil, LoginResult{}, nil, ErrLoginPasswordRequired
	}

	client, err := newLoginClient(opts...)
	if err != nil {
		return nil, LoginResult{}, nil, err
	}
	startedAt := request.StartedAt
	if startedAt.IsZero() {
		startedAt = time.Now()
	}
	payload := struct {
		Identifier     string `json:"identifier"`
		Password       string `json:"password"`
		Nickname       string `json:"nickname"`
		StartedAt      int64  `json:"startedAt"`
		Automated      bool   `json:"automated"`
		TurnstileToken string `json:"turnstileToken,omitempty"`
		AddAccount     bool   `json:"add,omitempty"`
	}{
		Identifier:     request.Identifier,
		Password:       request.Password,
		StartedAt:      startedAt.UnixMilli(),
		TurnstileToken: strings.TrimSpace(request.TurnstileToken),
		AddAccount:     request.AddAccount,
	}

	result, response, err := doJSON[LoginResult](ctx, client, http.MethodPost, "/api/auth/login", nil, payload)
	if err != nil {
		return nil, LoginResult{}, response, err
	}
	if result.RequiresTwoFactor {
		if strings.TrimSpace(result.TwoFactorToken) == "" {
			return nil, LoginResult{}, response, ErrTwoFactorTokenRequired
		}

		return client, result, response, nil
	}
	if err := client.verifySession(ctx); err != nil {
		return nil, LoginResult{}, response, err
	}

	return client, result, response, nil
}

// CompleteTwoFactor finishes a login using an authenticator or backup code.
func (c *Client) CompleteTwoFactor(ctx context.Context, token, code string) (*Response, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrTwoFactorTokenRequired
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, ErrTwoFactorCodeRequired
	}

	response, err := doEmpty(ctx, c, http.MethodPost, "/api/auth/login/2fa", nil, struct {
		Token string `json:"token"`
		Code  string `json:"code"`
	}{Token: token, Code: code})
	if err != nil {
		return response, err
	}
	if err := c.verifySession(ctx); err != nil {
		return response, err
	}

	return response, nil
}

// Me returns the account attached to the current web session.
func (c *Client) Me(ctx context.Context) (User, *Response, error) {
	data, response, err := doJSON[struct {
		User User `json:"user"`
	}](ctx, c, http.MethodGet, "/api/auth/me", nil, nil)
	if err != nil {
		return User{}, response, err
	}
	if data.User.ID == "" {
		return User{}, response, ErrNotAuthenticated
	}

	return data.User, response, nil
}

// Logout ends the current web session.
func (c *Client) Logout(ctx context.Context) (*Response, error) {
	return doEmpty(ctx, c, http.MethodPost, "/api/auth/logout", nil, nil)
}

func (c *Client) verifySession(ctx context.Context) error {
	if c.CookieHeader() == "" {
		return ErrSessionNotEstablished
	}
	if _, _, err := c.Me(ctx); err != nil {
		return errors.Join(ErrSessionNotEstablished, err)
	}

	return nil
}

func newLoginClient(opts ...Option) (*Client, error) {
	cfg, baseURL, err := configure(opts)
	if err != nil {
		return nil, err
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	if cfg.httpClient.Jar != nil {
		target := apiURL(baseURL)
		jar.SetCookies(target, cfg.httpClient.Jar.Cookies(target))
	}
	httpClient := *cfg.httpClient
	httpClient.Jar = jar
	cfg.httpClient = &httpClient

	return configuredClient(cfg, baseURL, webHeaders(baseURL)), nil
}
