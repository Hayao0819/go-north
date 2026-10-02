package oauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

type seenRequest struct {
	path string
	form url.Values
}

func writeJSON(t *testing.T, writer http.ResponseWriter, status int, body string) {
	t.Helper()

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	if _, err := writer.Write([]byte(body)); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

func TestNewConfig(t *testing.T) {
	t.Parallel()

	config := NewConfig("client", "http://127.0.0.1/callback", "posts.read", "users.read")
	if config.ClientID != "client" || config.ClientSecret != "" {
		t.Fatalf("client credentials = %q, %q", config.ClientID, config.ClientSecret)
	}
	if config.RedirectURL != "http://127.0.0.1/callback" || len(config.Scopes) != 2 {
		t.Fatalf("config = %#v", config)
	}
	if config.Endpoint != Endpoint || config.Endpoint.AuthStyle != oauth2.AuthStyleInParams {
		t.Fatalf("endpoint = %#v", config.Endpoint)
	}
}

func TestOAuthRequests(t *testing.T) {
	t.Parallel()

	seen := make(chan seenRequest, 8)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if err := request.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
		}
		seen <- seenRequest{
			path: request.URL.Path,
			form: request.Form,
		}
		if request.Header.Get("Authorization") != "" {
			t.Errorf("OAuth request has Authorization header")
		}
		if request.Form.Has("client_secret") {
			t.Error("OAuth request has client_secret")
		}

		switch request.URL.Path {
		case "/oauth/device/code":
			writeJSON(t, writer, http.StatusOK, `{"device_code":"device","user_code":"ABCD","verification_uri":"https://north.rip/device","expires_in":600,"interval":5}`)
		case "/oauth/token":
			writeJSON(t, writer, http.StatusOK, `{"access_token":"nth_oat_access","refresh_token":"rotated","token_type":"Bearer","expires_in":3600,"scope":"posts.read users.read"}`)
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	config := NewConfig("client", "http://127.0.0.1/callback", "posts.read", "users.read")
	config.Endpoint = oauth2.Endpoint{
		AuthURL:       server.URL + "/oauth/authorize",
		DeviceAuthURL: server.URL + "/oauth/device/code",
		TokenURL:      server.URL + "/oauth/token",
		AuthStyle:     oauth2.AuthStyleInParams,
	}
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, server.Client())

	authorization, err := config.DeviceAuth(ctx)
	if err != nil || authorization.DeviceCode != "device" {
		t.Fatalf("DeviceAuth = %#v, %v", authorization, err)
	}
	request := <-seen
	if request.path != "/oauth/device/code" || request.form.Get("client_id") != "client" || request.form.Get("scope") != "posts.read users.read" {
		t.Errorf("device request = %#v", request)
	}
	authorization.Interval = 1
	token, err := config.DeviceAccessToken(ctx, authorization)
	if err != nil || token.AccessToken != "nth_oat_access" {
		t.Fatalf("DeviceAccessToken = %#v, %v", token, err)
	}
	request = <-seen
	if request.form.Get("grant_type") != "urn:ietf:params:oauth:grant-type:device_code" || request.form.Get("device_code") != "device" || request.form.Get("client_id") != "client" {
		t.Errorf("device exchange request = %#v", request)
	}

	verifier := oauth2.GenerateVerifier()
	token, err = config.Exchange(ctx, "code", oauth2.VerifierOption(verifier))
	if err != nil || token.AccessToken != "nth_oat_access" || token.Extra("scope") != "posts.read users.read" {
		t.Fatalf("Exchange = %#v, %v", token, err)
	}
	request = <-seen
	if request.form.Get("grant_type") != "authorization_code" || request.form.Get("code_verifier") != verifier || request.form.Get("client_id") != "client" {
		t.Errorf("exchange request = %#v", request)
	}

	token, err = config.TokenSource(ctx, &oauth2.Token{RefreshToken: "old-refresh"}).Token()
	if err != nil || token.RefreshToken != "rotated" {
		t.Fatalf("refresh = %#v, %v", token, err)
	}
	request = <-seen
	if request.form.Get("grant_type") != "refresh_token" || request.form.Get("refresh_token") != "old-refresh" || request.form.Get("client_id") != "client" {
		t.Errorf("refresh request = %#v", request)
	}
}

func TestTokenError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(t, writer, http.StatusBadRequest, `{"error":"invalid_grant","error_description":"expired"}`)
	}))
	defer server.Close()
	config := NewConfig("client", "http://127.0.0.1/callback")
	config.Endpoint.TokenURL = server.URL
	ctx := context.WithValue(context.Background(), oauth2.HTTPClient, server.Client())

	_, err := config.Exchange(ctx, "code", oauth2.VerifierOption(strings.Repeat("v", 43)))
	var retrieveError *oauth2.RetrieveError
	if !errors.As(err, &retrieveError) || retrieveError.ErrorCode != "invalid_grant" {
		t.Fatalf("Exchange error = %#v", err)
	}
}
