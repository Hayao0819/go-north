package north

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

// MediaKind is the kind of media attached to a post.
type MediaKind string

const (
	MediaPhoto MediaKind = "PHOTO"
	MediaGIF   MediaKind = "GIF"
	MediaVideo MediaKind = "VIDEO"
)

// MediaStatus is the server-side processing state of uploaded media.
type MediaStatus string

const (
	MediaPending MediaStatus = "PENDING"
	MediaReady   MediaStatus = "READY"
	MediaFailed  MediaStatus = "FAILED"
)

// MediaWarning is the sensitive-content warning on an attachment.
type MediaWarning string

const (
	WarningNudity    MediaWarning = "NUDITY"
	WarningViolence  MediaWarning = "VIOLENCE"
	WarningSensitive MediaWarning = "SENSITIVE"
)

// Media is an uploaded attachment.
type Media struct {
	ID           string        `json:"id"`
	Kind         MediaKind     `json:"kind"`
	Status       MediaStatus   `json:"status"`
	URL          string        `json:"url"`
	ThumbnailURL *string       `json:"thumbnailUrl"`
	Width        int           `json:"width"`
	Height       int           `json:"height"`
	DurationMS   *int64        `json:"durationMs"`
	AltText      *string       `json:"altText"`
	Sensitive    bool          `json:"sensitive"`
	Warning      *MediaWarning `json:"warning"`
}

// UploadSession starts a chunked media upload.
type UploadSession struct {
	UploadID  string `json:"uploadId"`
	ChunkSize int64  `json:"chunkSize"`
}

// UploadProgress reports the cumulative bytes accepted for a chunked upload.
type UploadProgress struct {
	Received int64 `json:"received"`
}

// UploadMedia uploads one image, GIF, or video with multipart/form-data.
// Files over 100 MB must use the chunked upload methods instead.
func (c *Client) UploadMedia(ctx context.Context, filename string, content io.Reader) (Media, *Response, error) {
	if content == nil {
		return Media{}, nil, errors.New("north: media content must not be nil")
	}

	name := filepath.Base(filename)
	if name == "." || name == string(filepath.Separator) || name == "" {
		name = "upload"
	}

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	go func() {
		part, err := mw.CreateFormFile("file", name)
		if err == nil {
			_, err = io.Copy(part, content)
		}
		if closeErr := mw.Close(); err == nil {
			err = closeErr
		}
		_ = pw.CloseWithError(err)
	}()

	var media Media
	resp, err := c.do(ctx, http.MethodPost, "/2/media/upload", nil, pr, mw.FormDataContentType(), &media)
	if err != nil {
		return Media{}, resp, err
	}

	return media, resp, nil
}

// UploadMediaFile opens path and uploads it. The file is closed before the
// method returns.
func (c *Client) UploadMediaFile(ctx context.Context, path string) (Media, *Response, error) {
	f, err := os.Open(path)
	if err != nil {
		return Media{}, nil, fmt.Errorf("north: open media: %w", err)
	}
	defer f.Close()

	return c.UploadMedia(ctx, filepath.Base(path), f)
}

// InitializeUpload begins a chunked upload and returns the chunk size the
// server expects.
func (c *Client) InitializeUpload(ctx context.Context, size int64, mediaType string) (UploadSession, *Response, error) {
	req := struct {
		Size      int64  `json:"size"`
		MediaType string `json:"mimetype"`
	}{Size: size, MediaType: mediaType}

	body, contentType, err := jsonRequest(req)
	if err != nil {
		return UploadSession{}, nil, err
	}

	var session UploadSession
	resp, err := c.do(ctx, http.MethodPost, "/2/media/upload/initialize", nil, body, contentType, &session)

	return session, resp, err
}

// AppendUpload sends the next chunk. Chunks must be sent in order and should
// follow the size returned by InitializeUpload.
func (c *Client) AppendUpload(ctx context.Context, uploadID string, chunk []byte) (UploadProgress, *Response, error) {
	var progress UploadProgress
	resp, err := c.do(
		ctx,
		http.MethodPost,
		"/2/media/upload/"+url.PathEscape(uploadID)+"/append",
		nil,
		bytes.NewReader(chunk),
		"application/octet-stream",
		&progress,
	)

	return progress, resp, err
}

// FinalizeUpload completes a chunked upload. Video media may remain PENDING
// while north converts it.
func (c *Client) FinalizeUpload(ctx context.Context, uploadID string) (Media, *Response, error) {
	var media Media
	resp, err := c.do(ctx, http.MethodPost, "/2/media/upload/"+url.PathEscape(uploadID)+"/finalize", nil, nil, "", &media)

	return media, resp, err
}

// SetMediaAltText adds or replaces alternative text on media owned by the
// caller.
func (c *Client) SetMediaAltText(ctx context.Context, mediaID, text string) (bool, *Response, error) {
	req := struct {
		AltText struct {
			Text string `json:"text"`
		} `json:"alt_text"`
	}{}
	req.AltText.Text = text

	body, contentType, err := jsonRequest(req)
	if err != nil {
		return false, nil, err
	}

	data, resp, err := doData[struct {
		OK bool `json:"ok"`
	}](ctx, c, http.MethodPut, "/2/media/"+url.PathEscape(mediaID)+"/metadata", nil, body, contentType)

	return data.OK, resp, err
}
