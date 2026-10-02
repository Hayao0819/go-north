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

// MediaPurpose tells north where an upload will be used.
type MediaPurpose string

const (
	MediaForPosts   MediaPurpose = "posts"
	MediaForProfile MediaPurpose = "profile"
	MediaForDraft   MediaPurpose = "draft"
	MediaForDM      MediaPurpose = "dm"
)

// MediaUploadOption configures a media upload request.
type MediaUploadOption func(http.Header)

// WithMediaPurpose sets the destination of an upload.
func WithMediaPurpose(purpose MediaPurpose) MediaUploadOption {
	return func(header http.Header) {
		header.Set("X-North-Media-Purpose", string(purpose))
	}
}

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
	Size     int64 `json:"size"`
}

// UploadMedia uploads one image, GIF, or video with multipart/form-data.
// Files over 100 MB must use the chunked upload methods instead.
func (c *Client) UploadMedia(ctx context.Context, filename string, content io.Reader, opts ...MediaUploadOption) (Media, *Response, error) {
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

	return doDataOrValue[Media](ctx, c, http.MethodPost, "/2/media/upload", nil, pr, mw.FormDataContentType(), mediaHeader(opts))
}

// UploadMediaFile opens path and uploads it. The file is closed before the
// method returns.
func (c *Client) UploadMediaFile(ctx context.Context, path string, opts ...MediaUploadOption) (Media, *Response, error) {
	f, err := os.Open(path)
	if err != nil {
		return Media{}, nil, fmt.Errorf("north: open media: %w", err)
	}
	defer f.Close()

	return c.UploadMedia(ctx, filepath.Base(path), f, opts...)
}

// InitializeUpload begins a chunked upload and returns the chunk size the
// server expects.
func (c *Client) InitializeUpload(ctx context.Context, size int64, mediaType string, opts ...MediaUploadOption) (UploadSession, *Response, error) {
	req := struct {
		Size      int64  `json:"size"`
		MediaType string `json:"mimetype"`
	}{Size: size, MediaType: mediaType}

	body, contentType, err := jsonRequest(req)
	if err != nil {
		return UploadSession{}, nil, err
	}

	return doDataOrValue[UploadSession](ctx, c, http.MethodPost, "/2/media/upload/initialize", nil, body, contentType, mediaHeader(opts))
}

// AppendUpload sends the next chunk. Chunks must be sent in order and should
// follow the size returned by InitializeUpload.
func (c *Client) AppendUpload(ctx context.Context, uploadID string, chunk []byte, opts ...MediaUploadOption) (UploadProgress, *Response, error) {
	return doDataOrValue[UploadProgress](
		ctx,
		c,
		http.MethodPost,
		"/2/media/upload/"+url.PathEscape(uploadID)+"/append",
		nil,
		bytes.NewReader(chunk),
		"application/octet-stream",
		mediaHeader(opts),
	)
}

// FinalizeUpload completes a chunked upload. Video media may remain PENDING
// while north converts it.
func (c *Client) FinalizeUpload(ctx context.Context, uploadID string, opts ...MediaUploadOption) (Media, *Response, error) {
	return doDataOrValue[Media](ctx, c, http.MethodPost, "/2/media/upload/"+url.PathEscape(uploadID)+"/finalize", nil, nil, "", mediaHeader(opts))
}

// CancelUpload cancels a chunked upload.
func (c *Client) CancelUpload(ctx context.Context, uploadID string) (bool, *Response, error) {
	return doOK(ctx, c, http.MethodDelete, "/2/media/upload/"+url.PathEscape(uploadID), nil, nil, "")
}

// Media returns media owned by the caller.
func (c *Client) Media(ctx context.Context, mediaID string) (Media, *Response, error) {
	return doData[Media](ctx, c, http.MethodGet, "/2/media/"+url.PathEscape(mediaID), nil, nil, "")
}

// DeleteMedia deletes unattached media owned by the caller.
func (c *Client) DeleteMedia(ctx context.Context, mediaID string) (bool, *Response, error) {
	return doOK(ctx, c, http.MethodDelete, "/2/media/"+url.PathEscape(mediaID), nil, nil, "")
}

// SetMediaWarning sets or clears a media content warning. Nil clears it.
func (c *Client) SetMediaWarning(ctx context.Context, mediaID string, warning *MediaWarning) (*MediaWarning, *Response, error) {
	body, contentType, err := jsonRequest(struct {
		Warning *MediaWarning `json:"warning"`
	}{Warning: warning})
	if err != nil {
		return nil, nil, err
	}
	data, response, err := doData[struct {
		Warning *MediaWarning `json:"warning"`
	}](ctx, c, http.MethodPut, "/2/media/"+url.PathEscape(mediaID)+"/warning", nil, body, contentType)

	return data.Warning, response, err
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

func mediaHeader(opts []MediaUploadOption) http.Header {
	header := make(http.Header)
	for _, opt := range opts {
		if opt != nil {
			opt(header)
		}
	}

	return header
}
