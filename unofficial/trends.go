package unofficial

import (
	"context"
	"net/http"
)

// Trend is one entry in north's current trend list.
type Trend struct {
	Tag       string `json:"tag"`
	Count     int    `json:"count"`
	IsHashtag bool   `json:"isHashtag"`
}

// TrendList contains the trends returned by the web application.
type TrendList struct {
	Items []Trend `json:"items"`
}

// Trends returns the current trends. This route does not require a web
// session, so it can be called with a client created by NewPublicClient.
func (c *Client) Trends(ctx context.Context) (TrendList, *Response, error) {
	return doJSON[TrendList](ctx, c, http.MethodGet, "/api/trends", nil, nil)
}
