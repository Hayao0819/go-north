package north

import (
	"context"
	"net/http"
	"net/url"
)

// Trend is one entry in the caller's trend list.
type Trend struct {
	Tag       string `json:"tag"`
	Count     int    `json:"count"`
	IsHashtag bool   `json:"isHashtag"`
}

// Trends returns the caller's trend list.
func (c *Client) Trends(ctx context.Context, cursor string) ([]Trend, *Response, error) {
	query := url.Values{}
	setCursor(query, cursor)
	data, response, err := doData[struct {
		Items []Trend `json:"items"`
	}](ctx, c, http.MethodGet, "/2/trends", query, nil, "")

	return data.Items, response, err
}

// DismissTrend hides a trend from the caller's list.
func (c *Client) DismissTrend(ctx context.Context, tag string) (bool, *Response, error) {
	body, contentType, err := jsonRequest(struct {
		Tag string `json:"tag"`
	}{Tag: tag})
	if err != nil {
		return false, nil, err
	}

	return doOK(ctx, c, http.MethodPost, "/2/trends/dismiss", nil, body, contentType)
}
