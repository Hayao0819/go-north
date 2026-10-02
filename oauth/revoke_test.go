package oauth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/oauth2"
)

func TestRevoke(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if err := request.ParseForm(); err != nil {
			t.Errorf("ParseForm: %v", err)
		}
		if request.Form.Get("client_id") != "client" || request.Form.Get("token") != "token" {
			t.Errorf("revoke form = %#v", request.Form)
		}
		writeJSON(t, writer, http.StatusOK, `{}`)
	}))
	defer server.Close()

	if err := Revoke(context.Background(), server.Client(), server.URL, "client", "token"); err != nil {
		t.Fatal(err)
	}
}

func TestRevokeError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writeJSON(t, writer, http.StatusBadRequest, `{"error":"invalid_grant","error_description":"expired"}`)
	}))
	defer server.Close()

	err := Revoke(context.Background(), server.Client(), server.URL, "client", "token")
	var retrieveError *oauth2.RetrieveError
	if !errors.As(err, &retrieveError) || retrieveError.ErrorCode != "invalid_grant" || retrieveError.ErrorDescription != "expired" {
		t.Fatalf("Revoke error = %#v", err)
	}
}
