package north

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestMediaEndpoints(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/api/2/media/upload":
			if err := request.ParseMultipartForm(1 << 20); err != nil {
				t.Errorf("ParseMultipartForm: %v", err)
			}
			file, header, err := request.FormFile("file")
			if err != nil {
				t.Errorf("FormFile: %v", err)
			} else {
				defer file.Close()
				content, _ := io.ReadAll(file)
				if header.Filename != "photo.jpg" || string(content) != "jpeg-data" {
					t.Errorf("upload = %q %q", header.Filename, content)
				}
			}
			writeJSON(t, writer, http.StatusCreated, mediaJSON("media-1"))
		case "/api/2/media/upload/initialize":
			var body struct {
				Size     int64  `json:"size"`
				MimeType string `json:"mimetype"`
			}
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode initialize: %v", err)
			}
			if body.Size != 200_000_000 || body.MimeType != "video/mp4" {
				t.Errorf("initialize = %#v", body)
			}
			writeJSON(t, writer, http.StatusCreated, `{"uploadId":"upload-1","chunkSize":4}`)
		case "/api/2/media/upload/upload-1/append":
			body, _ := io.ReadAll(request.Body)
			if request.Header.Get("Content-Type") != "application/octet-stream" || !bytes.Equal(body, []byte{1, 2, 3, 4}) {
				t.Errorf("append = %q %v", request.Header.Get("Content-Type"), body)
			}
			writeJSON(t, writer, http.StatusOK, `{"received":4}`)
		case "/api/2/media/upload/upload-1/finalize":
			writeJSON(t, writer, http.StatusCreated, mediaJSON("media-2"))
		case "/api/2/media/media-2/metadata":
			var body struct {
				AltText struct {
					Text string `json:"text"`
				} `json:"alt_text"`
			}
			if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
				t.Errorf("decode metadata: %v", err)
			}
			if body.AltText.Text != "sunset" {
				t.Errorf("alt text = %q", body.AltText.Text)
			}
			writeJSON(t, writer, http.StatusOK, `{"data":{"ok":true}}`)
		default:
			http.NotFound(writer, request)
		}
	})

	ctx := context.Background()
	media, _, err := client.UploadMedia(ctx, "/tmp/photo.jpg", bytes.NewBufferString("jpeg-data"))
	if err != nil || media.ID != "media-1" {
		t.Fatalf("UploadMedia = %#v, %v", media, err)
	}

	session, _, err := client.InitializeUpload(ctx, 200_000_000, "video/mp4")
	if err != nil || session.UploadID != "upload-1" || session.ChunkSize != 4 {
		t.Fatalf("InitializeUpload = %#v, %v", session, err)
	}

	progress, _, err := client.AppendUpload(ctx, session.UploadID, []byte{1, 2, 3, 4})
	if err != nil || progress.Received != 4 {
		t.Fatalf("AppendUpload = %#v, %v", progress, err)
	}

	media, _, err = client.FinalizeUpload(ctx, session.UploadID)
	if err != nil || media.ID != "media-2" {
		t.Fatalf("FinalizeUpload = %#v, %v", media, err)
	}

	ok, _, err := client.SetMediaAltText(ctx, media.ID, "sunset")
	if err != nil || !ok {
		t.Fatalf("SetMediaAltText = %v, %v", ok, err)
	}
}

func TestUploadMediaRejectsNilReader(t *testing.T) {
	t.Parallel()

	client, err := NewClient("token")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := client.UploadMedia(context.Background(), "file", nil); err == nil {
		t.Fatal("UploadMedia accepted a nil reader")
	}
}

func mediaJSON(id string) string {
	return `{"id":"` + id + `","kind":"PHOTO","status":"READY","url":"https://cdn.example/1","width":640,"height":480}`
}
