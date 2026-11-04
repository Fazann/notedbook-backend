package dto

import "github.com/Fazann/notedbook-backend/pkg/enums"

type FileData struct {
	Filename     string
	Size         int64
	Width        int16
	Height       int16
	OriginalName string
	Mimetype     string // detected from the file's content, not the client's header
	Path         string
	UploadedBy   uint
}

type AttachmentRes struct {
	ID           uint                     `json:"id"`
	Type         enums.AttachmentTypeEnum `json:"type"`
	Filename     string                   `json:"filename"`
	Size         int64                    `json:"size"`
	Width        int16                    `json:"width"`
	Height       int16                    `json:"height"`
	OriginalName string                   `json:"original_name"`
	MimeType     string                   `json:"mimetype"`
	Url          string                   `json:"path"`
	ThumbnailUrl string                   `json:"thumbnail_url"`
	Duration     *float64                 `json:"duration"`
}
