package repo

import (
	"context"
	"time"

	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/shared/baserepo"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

type AttachmentRepoImpl interface {
	baserepo.BaseRepo[model.Attachment]
	MarkUsed(ctx context.Context, id uint, userID uint, usedAt time.Time) (bool, error)
	SoftDelete(ctx context.Context, id uint, deletedBy uint) error
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

// MarkUsed sets used_at on an attachment of userID that is not used yet. It reports false when the
// attachment is already used (or does not exist, or is another user's), so two records cannot claim
// the same file.
func (r *attachmentRepo) MarkUsed(ctx context.Context, id uint, userID uint, usedAt time.Time) (bool, error) {
	res := r.db.WithContext(ctx).Model(&model.Attachment{}).
		Where("id = ? AND created_by = ? AND used_at IS NULL", id, userID).
		Update("used_at", usedAt)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected == 1, nil
}

// SoftDelete marks an attachment as deleted by deletedBy; the file in S3 is kept.
func (r *attachmentRepo) SoftDelete(ctx context.Context, id uint, deletedBy uint) error {
	return r.db.WithContext(ctx).Model(&model.Attachment{}).
		Where("id = ?", id).
		Updates(map[string]any{"deleted_at": time.Now(), "deleted_by": deletedBy}).Error
}
