package handler

import (
	"net/http"
	"strconv"

	"fmt"
	"path/filepath"
	"time"

	"github.com/Fazann/notedbook-backend/internal/attachments/dto"
	"github.com/Fazann/notedbook-backend/internal/attachments/service"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/response"
	"github.com/Fazann/notedbook-backend/pkg/utils"
	"github.com/gin-gonic/gin"
)

type AttachmentHandler struct {
	attachmentSvc service.AttachmentServiceImpl
}

const (
	MaxUploadSize = 100 << 20 // 100 MB
)

func NewAttachmentHandler() *AttachmentHandler {
	return &AttachmentHandler{
		attachmentSvc: service.NewAttachmentSvc(),
	}
}

// GetDetail godoc
//
//	@Summary		Get an attachment
//	@Description	Returns one attachment by ID, with its full file URL.
//	@Tags			attachments
//	@Produce		json
//	@Param			id				path		int		true	"Attachment ID"					minimum(1)
//	@Param			Accept-Language	header		string	false	"Language of error messages"	Enums(en, km, ms)
//	@Success		200				{object}	response.SuccessRes{data=dto.AttachmentRes}
//	@Failure		400				{object}	response.ErrorRes	"INVALID_ID"
//	@Failure		404				{object}	response.ErrorRes	"RECORD_NOT_FOUND"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Failure		500				{object}	response.ErrorRes	"INTERNAL_SERVER_ERROR"
//	@Router			/v1/attachments/{id} [get]
func (h *AttachmentHandler) GetDetail(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if id <= 0 {
		response.JsonErrRes(c, i18n.InvalidID, http.StatusBadRequest)
		return
	}

	res := h.attachmentSvc.GetDetail(c.Request.Context(), uint(id))
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}

// UploadFile godoc
//
//	@Summary		Upload a file
//	@Description	Uploads a file (max 100 MB) to storage and saves it as an attachment.
//	@Description	The type (image, audio, video, file) is detected from the file's MIME type.
//	@Tags			attachments
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			file			formData	file	true	"File to upload"
//	@Param			Accept-Language	header		string	false	"Language of error messages"	Enums(en, km, ms)
//	@Success		201				{object}	response.SuccessRes{data=dto.AttachmentRes}
//	@Failure		400				{object}	response.ErrorRes	"FILE_REQUIRED or BAD_REQUEST"
//	@Failure		413				{object}	response.ErrorRes	"FILE_TOO_LARGE"
//	@Failure		429				{object}	response.ErrorRes	"TOO_MANY_REQUESTS"
//	@Failure		500				{object}	response.ErrorRes	"INTERNAL_SERVER_ERROR"
//	@Router			/v1/attachments [post]
func (h *AttachmentHandler) UploadFile(c *gin.Context) {
	// Wrap the body reader so only MaxUploadSize bytes are allowed
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxUploadSize)
	// Parse multipart form
	if err := c.Request.ParseMultipartForm(MaxUploadSize); err != nil {
		if _, ok := err.(*http.MaxBytesError); ok {
			response.JsonErrRes(
				c,
				i18n.Err(i18n.FileTooLarge, i18n.Params{"max": fmt.Sprintf("%d MB", MaxUploadSize>>20)}),
				http.StatusRequestEntityTooLarge,
			)
			return
		}
		response.JsonErrRes(c, err, http.StatusBadRequest)
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		response.JsonErrRes(c, i18n.FileRequired, http.StatusBadRequest)
		return
	}

	filename := fmt.Sprintf(
		"img-%d%s",
		time.Now().UnixNano(),
		filepath.Ext(file.Filename),
	)
	dst := filepath.Join(utils.GetTempFilePath(), filename)
	err = c.SaveUploadedFile(file, dst)
	if err != nil {
		response.JsonErrRes(c, err, http.StatusInternalServerError)
		return
	}

	fileData := dto.FileData{
		Filename:     filename,
		Size:         file.Size,
		Width:        0,
		Height:       0,
		OriginalName: file.Filename,
		Mimetype:     file.Header.Get("Content-Type"),
		Path:         dst,
	}

	res := h.attachmentSvc.UploadFile(c.Request.Context(), fileData)
	if res.Error != nil {
		response.JsonErrRes(c, res.Error, res.StatusCode)
		return
	}

	response.JsonSuccessRes(c, res.Data, res.StatusCode)
}
