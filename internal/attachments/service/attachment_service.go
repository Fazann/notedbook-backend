package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"net/http"
	"os"

	"github.com/Fazann/notedbook-backend/internal/attachments/dto"
	"github.com/Fazann/notedbook-backend/internal/attachments/repo"
	"github.com/Fazann/notedbook-backend/pkg/enums"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/logger"
	"github.com/Fazann/notedbook-backend/pkg/response"
	"github.com/Fazann/notedbook-backend/pkg/utils"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

type AttachmentServiceImpl interface {
	GetDetail(ctx context.Context, id uint, userID uint) response.SvcRes
	UploadFile(ctx context.Context, file dto.FileData) response.SvcRes
	MarkUsed(ctx context.Context, id uint, userID uint, types ...enums.AttachmentTypeEnum) error
	Delete(ctx context.Context, id uint, deletedBy uint) error
}

type attachmentSvc struct {
	repo repo.AttachmentRepoImpl
}

var imageMimetype = []string{
	"image/jpeg",
	"image/png",
	"image/gif",
	"image/webp",
	"image/bmp",
	"image/tiff",
	"image/x-icon",
	"image/vnd.microsoft.icon",
}

var audioMimetype = []string{
	"audio/mpeg",  // MP3
	"audio/wav",   // WAV
	"audio/wave",  // alternative WAV
	"audio/x-wav", // legacy WAV
	"audio/ogg",   // OGG audio
	"audio/webm",  // WebM audio
	"audio/flac",  // FLAC
}

var videoMimetype = []string{
	"video/mp4",        // MP4 (most common)
	"video/webm",       // WebM (modern web video)
	"video/ogg",        // OGG video
	"video/avi",        // AVI, as http.DetectContentType names it
	"video/x-msvideo",  // AVI
	"video/x-matroska", // MKV
}

// AllowedMimetypes lists what can be uploaded, for error messages.
const AllowedMimetypes = "image, audio, video"

// IsAllowedMimetype reports whether a detected MIME type may be uploaded (images, audio and video).
func IsAllowedMimetype(mimetype string) bool {
	return utils.IsInclude(imageMimetype, mimetype) ||
		utils.IsInclude(audioMimetype, mimetype) ||
		utils.IsInclude(videoMimetype, mimetype)
}

// extByMimetype names stored files by their detected type, never by the client's file name.
var extByMimetype = map[string]string{
	"image/jpeg": ".jpg", "image/png": ".png", "image/gif": ".gif", "image/webp": ".webp",
	"image/bmp": ".bmp", "image/x-icon": ".ico",
	"audio/mpeg": ".mp3", "audio/wave": ".wav", "audio/wav": ".wav", "audio/x-wav": ".wav",
	"video/mp4": ".mp4", "video/webm": ".webm", "video/avi": ".avi",
}

// ExtForMimetype returns the file extension for a detected MIME type ("" when unknown).
func ExtForMimetype(mimetype string) string {
	return extByMimetype[mimetype]
}

func NewAttachmentSvc() *attachmentSvc {
	return &attachmentSvc{
		repo: repo.NewAttachmentRepo(),
	}
}

// GetDetail returns an attachment uploaded by userID; another user's file is RECORD_NOT_FOUND.
func (s *attachmentSvc) GetDetail(ctx context.Context, id uint, userID uint) response.SvcRes {
	attachment, err := s.repo.FindOne(ctx, "id = ? AND created_by = ?", id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return response.SvcRespErr(i18n.RecordNotFound, http.StatusNotFound)
		}

		return response.SvcRespErr(err, http.StatusInternalServerError)
	}

	return response.SvcRes{
		Data: &dto.AttachmentRes{
			ID:           attachment.ID,
			Type:         attachment.Type,
			Filename:     attachment.Filename,
			OriginalName: attachment.OriginalName,
			Size:         attachment.Size,
			Height:       attachment.Height,
			Width:        attachment.Width,
			Url:          attachment.Path,
			ThumbnailUrl: attachment.ThumbnailUrl,
			Duration:     attachment.Duration,
			MimeType:     attachment.Mimetype,
		},
		StatusCode: 200,
	}
}

func (s *attachmentSvc) UploadFile(ctx context.Context, file dto.FileData) response.SvcRes {
	// 1. Upload file to s3
	s3Res, err := utils.UploadFile(file.Path, file.Filename, file.Mimetype)
	defer s.removeTempFile(file.Path)

	if err != nil {
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}

	// 2. Save to db
	data := model.Attachment{
		Type:         s.getFileTypeByMimetype(file.Mimetype),
		Mimetype:     file.Mimetype,
		Filename:     file.Filename,
		OriginalName: file.OriginalName,
		Size:         file.Size,
		Height:       file.Height,
		Width:        file.Width,
		Path:         s3Res.Path,
	}
	data.CreatedBy = file.UploadedBy
	attachment, err := s.repo.Create(ctx, &data)
	if err != nil {
		return response.SvcRespErr(err, http.StatusInternalServerError)
	}

	return response.SvcRes{
		Data: &dto.AttachmentRes{
			ID:           attachment.ID,
			Type:         attachment.Type,
			Filename:     attachment.Filename,
			OriginalName: attachment.OriginalName,
			MimeType:     attachment.Mimetype,
			Size:         attachment.Size,
			Height:       attachment.Height,
			Width:        attachment.Width,
			Url:          s3Res.URL,
		},
		StatusCode: 201,
	}
}

// MarkUsed marks an unused attachment uploaded by userID as used by a record (e.g. the user's avatar).
// When types are given, the attachment must be one of them. Errors: i18n.RecordNotFound (also for
// another user's file), i18n.InvalidAttachmentType, i18n.AttachmentInUse, or a database error.
func (s *attachmentSvc) MarkUsed(ctx context.Context, id uint, userID uint, types ...enums.AttachmentTypeEnum) error {
	attachment, err := s.repo.FindOne(ctx, "id = ? AND created_by = ?", id, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return i18n.RecordNotFound
		}
		return fmt.Errorf("find attachment: %w", err)
	}

	if len(types) > 0 && !utils.IsInclude(types, attachment.Type) {
		names := make([]string, len(types))
		for i, t := range types {
			names[i] = string(t)
		}
		return i18n.Err(i18n.InvalidAttachmentType, i18n.Params{"types": strings.Join(names, ", ")})
	}

	marked, err := s.repo.MarkUsed(ctx, id, userID, time.Now())
	if err != nil {
		return fmt.Errorf("mark attachment used: %w", err)
	}
	if !marked {
		return i18n.AttachmentInUse
	}

	return nil
}

// Delete soft-deletes an attachment that is no longer used (e.g. a replaced avatar).
func (s *attachmentSvc) Delete(ctx context.Context, id uint, deletedBy uint) error {
	if err := s.repo.SoftDelete(ctx, id, deletedBy); err != nil {
		return fmt.Errorf("delete attachment: %w", err)
	}
	return nil
}

// ============================================================
// Local functions
// ============================================================

func (s *attachmentSvc) getFileTypeByMimetype(mimetype string) enums.AttachmentTypeEnum {
	if utils.IsInclude(imageMimetype, mimetype) {
		return enums.AttachmentTypeImage
	} else if utils.IsInclude(audioMimetype, mimetype) {
		return enums.AttachmentTypeAudio
	} else if utils.IsInclude(videoMimetype, mimetype) {
		return enums.AttachmentTypeVideo
	}
	return enums.AttachmentTypeFile
}

func (s *attachmentSvc) removeTempFile(filePath string) {
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		logger.Error("check temp file", "err", err, "path", filePath)
		return
	}

	err = os.Remove(filePath)
	if err != nil {
		logger.Error("remove temp file", "err", err, "path", filePath)
		return
	}
}
