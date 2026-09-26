package repo

import (
	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/shared/baserepo"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

type AttachmentRepoImpl interface {
	baserepo.BaseRepo[model.Attachment]
}

type attachmentRepo struct {
	db *gorm.DB
	baserepo.BaseRepo[model.Attachment]
}

func NewAttachmentRepo() *attachmentRepo {
	return &attachmentRepo{
		db:       db.DB,
		BaseRepo: baserepo.NewBaseRepo[model.Attachment](db.DB),
	}
}
