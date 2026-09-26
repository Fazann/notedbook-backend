package service

import (
	"context"
	"fmt"

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
	GetDetail(ctx context.Context, id uint) response.SvcRes
	UploadFile(ctx context.Context, file dto.FileData) response.SvcRes
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
	"video/x-msvideo",  // AVI
	"video/x-matroska", // MKV
}

func NewAttachmentSvc() *attachmentSvc {
	return &attachmentSvc{
		repo: repo.NewAttachmentRepo(),
	}
}

func (s *attachmentSvc) GetDetail(ctx context.Context, id uint) response.SvcRes {
	fmt.Printf("id %d \n", id)
	attachment, err := s.repo.FindOne(ctx, "id = ?", id)
	fmt.Printf("attachment %+v", attachment)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return response.SvcRespErr(i18n.RecordNotFound, http.StatusNotFound)
		}

		return response.SvcRespErr(err, http.StatusInternalServerError)
	}

	logger.Info("file", attachment)

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
	s3Res, err := utils.UploadFile(file.Path, file.Filename)
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
