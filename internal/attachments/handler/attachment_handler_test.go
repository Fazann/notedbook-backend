package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"strings"
	"testing"

	"github.com/Fazann/notedbook-backend/internal/attachments/dto"
	"github.com/Fazann/notedbook-backend/pkg/enums"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/response"
	"github.com/Fazann/notedbook-backend/shared/middleware"
	"github.com/gin-gonic/gin"
)

type fakeAttachmentSvc struct {
	uploaded  *dto.FileData
	detailFor [2]uint // id, userID
}

func (f *fakeAttachmentSvc) GetDetail(_ context.Context, id uint, userID uint) response.SvcRes {
	f.detailFor = [2]uint{id, userID}
	return response.SvcRes{Data: &dto.AttachmentRes{ID: id}, StatusCode: http.StatusOK}
}

func (f *fakeAttachmentSvc) UploadFile(_ context.Context, file dto.FileData) response.SvcRes {
	f.uploaded = &file
	_ = os.Remove(file.Path)
	return response.SvcRes{Data: &dto.AttachmentRes{ID: 1}, StatusCode: http.StatusCreated}
}

func (f *fakeAttachmentSvc) MarkUsed(context.Context, uint, uint, ...enums.AttachmentTypeEnum) error {
	return nil
}

func (f *fakeAttachmentSvc) Delete(context.Context, uint, uint) error { return nil }

// pngBytes is the start of a real PNG file (enough for http.DetectContentType).
var pngBytes = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00")

func newRouter(svc *fakeAttachmentSvc, userID uint) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := &AttachmentHandler{attachmentSvc: svc}
	setUser := func(c *gin.Context) { c.Set("auth_user_id", userID) }
	r := gin.New()
	r.POST("/attachments", setUser, h.UploadFile)
	r.GET("/attachments/:id", setUser, h.GetDetail)
	return r
}

func uploadReq(t *testing.T, name, contentType string, content []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	hdr := textproto.MIMEHeader{}
	hdr.Set("Content-Disposition", `form-data; name="file"; filename="`+name+`"`)
	hdr.Set("Content-Type", contentType)
	part, err := w.CreatePart(hdr)
	if err != nil {
		t.Fatal(err)
	}
	part.Write(content)
	w.Close()
	req := httptest.NewRequest(http.MethodPost, "/attachments", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestUploadFile(t *testing.T) {
	t.Cleanup(func() { os.RemoveAll("tmp") })

	tests := []struct {
		name        string
		filename    string
		contentType string // what the client claims
		content     []byte
		wantStatus  int
		wantMime    string
		wantExt     string
	}{
		{name: "png", filename: "me.png", contentType: "image/png", content: pngBytes, wantStatus: http.StatusCreated, wantMime: "image/png", wantExt: ".png"},
		{name: "png named and labelled as html is stored as png", filename: "x.html", contentType: "text/html", content: pngBytes, wantStatus: http.StatusCreated, wantMime: "image/png", wantExt: ".png"},
		{name: "html labelled as image is rejected", filename: "x.png", contentType: "image/png", content: []byte("<html><script>alert(1)</script></html>"), wantStatus: http.StatusBadRequest},
		{name: "svg is rejected", filename: "x.svg", contentType: "image/svg+xml", content: []byte(`<svg xmlns="http://www.w3.org/2000/svg"></svg>`), wantStatus: http.StatusBadRequest},
		{name: "pdf is rejected", filename: "x.pdf", contentType: "application/pdf", content: []byte("%PDF-1.7\n"), wantStatus: http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeAttachmentSvc{}
			w := httptest.NewRecorder()
			newRouter(svc, 7).ServeHTTP(w, uploadReq(t, tt.filename, tt.contentType, tt.content))

			if w.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (%s)", w.Code, tt.wantStatus, w.Body.String())
			}
			if tt.wantStatus != http.StatusCreated {
				if svc.uploaded != nil {
					t.Fatal("rejected file reached the service")
				}
				var body struct{ Code string }
				json.Unmarshal(w.Body.Bytes(), &body)
				if body.Code != string(i18n.InvalidAttachmentType) {
					t.Fatalf("code = %q, want %q", body.Code, i18n.InvalidAttachmentType)
				}
				return
			}
			got := svc.uploaded
			if got.Mimetype != tt.wantMime || !strings.HasSuffix(got.Filename, tt.wantExt) {
				t.Fatalf("stored as %q / %q, want %q / *%s", got.Mimetype, got.Filename, tt.wantMime, tt.wantExt)
			}
			if got.UploadedBy != 7 {
				t.Fatalf("UploadedBy = %d, want 7", got.UploadedBy)
			}
		})
	}
}

func TestGetDetailPassesUser(t *testing.T) {
	svc := &fakeAttachmentSvc{}
	w := httptest.NewRecorder()
	newRouter(svc, 7).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/attachments/3", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if svc.detailFor != [2]uint{3, 7} {
		t.Fatalf("GetDetail(id, user) = %v, want [3 7]", svc.detailFor)
	}
}

// Uploads raise the 1 MB JSON body limit for themselves.
func TestUploadAboveJSONBodyLimit(t *testing.T) {
	t.Cleanup(func() { os.RemoveAll("tmp") })
	gin.SetMode(gin.TestMode)
	svc := &fakeAttachmentSvc{}
	h := &AttachmentHandler{attachmentSvc: svc}
	r := gin.New()
	r.Use(middleware.BodyLimit(1 << 20))
	r.POST("/attachments", func(c *gin.Context) { c.Set("auth_user_id", uint(7)) }, h.UploadFile)

	big := append(append([]byte{}, pngBytes...), make([]byte, 2<<20)...)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, uploadReq(t, "big.png", "image/png", big))

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201 (%s)", w.Code, w.Body.String())
	}
	if svc.uploaded == nil || svc.uploaded.Size != int64(len(big)) {
		t.Fatalf("uploaded = %+v, want size %d", svc.uploaded, len(big))
	}
}
