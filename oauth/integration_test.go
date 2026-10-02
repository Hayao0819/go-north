//go:build integration

package oauth

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestIntegrationDeviceAuthorization(t *testing.T) {
	clientID := os.Getenv("NORTH_CLIENT_ID")
	if clientID == "" {
		t.Skip("NORTH_CLIENT_ID is not set")
	}

	config := NewConfig(clientID, "", "posts.read")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	authorization, err := config.DeviceAuth(ctx)
	if err != nil {
		t.Fatalf("DeviceAuth: %v", err)
	}
	if authorization.DeviceCode == "" || authorization.UserCode == "" || authorization.VerificationURI == "" {
		t.Fatal("DeviceAuth returned an incomplete authorization")
	}
	if authorization.Expiry.IsZero() || authorization.Interval <= 0 {
		t.Fatal("DeviceAuth returned invalid timing information")
	}
}
