package unofficial

import (
	"context"
	"net/http"
)

// Trends returns the current trends. This route does not require a web
// session, so it can be called with a client created by NewPublicClient.
func (c *Client) Trends(ctx context.Context) (TrendList, *Response, error) {
	return doJSON[TrendList](ctx, c, http.MethodGet, "/api/trends", nil, nil)
}
