package routes

import (
	"github.com/Fazann/notedbook-backend/internal/attachments/handler"
	"github.com/gin-gonic/gin"
)

type handlers struct {
	attachmentHandler handler.AttachmentHandler
}

func newHandlers() *handlers {
	return &handlers{
		attachmentHandler: *handler.NewAttachmentHandler(),
	}
}

func initAPIRoute(r *gin.Engine) {
	api := r.Group("/api/v1")

	h := newHandlers()

	attachments := api.Group("/attachments")
	attachments.POST("", h.attachmentHandler.UploadFile)
	attachments.GET("/:id", h.attachmentHandler.GetDetail)
}
