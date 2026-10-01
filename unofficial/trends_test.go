package unofficial

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPublicClientGetsTrendsWithoutACookie(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/api/trends" {
			t.Errorf("request = %s %s", request.Method, request.URL.Path)
		}
		if cookie := request.Header.Get("Cookie"); cookie != "" {
			t.Errorf("Cookie = %q", cookie)
		}
		writeJSON(t, writer, http.StatusOK, `{"items":[{"tag":"Go","count":12,"isHashtag":true}]}`)
	}))
	t.Cleanup(server.Close)

	client, err := NewPublicClient(WithBaseURL(server.URL), WithHTTPClient(server.Client()))
	if err != nil {
		t.Fatal(err)
	}
	trends, response, err := client.Trends(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || len(trends.Items) != 1 {
		t.Fatalf("Trends = %#v, response = %#v", trends, response)
	}
	if got := trends.Items[0]; got.Tag != "Go" || got.Count != 12 || !got.IsHashtag {
		t.Fatalf("trend = %#v", got)
	}
}
