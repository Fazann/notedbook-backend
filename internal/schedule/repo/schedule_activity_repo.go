package repo

import (
	"context"
	"time"

	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/pkg/enums"
	"github.com/Fazann/notedbook-backend/shared/baserepo"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ScheduleActivityRepoImpl interface {
	baserepo.BaseRepo[model.ScheduleActivity]
	WithTx(tx *gorm.DB) ScheduleActivityRepoImpl
	FindOneByID(ctx context.Context, userID, id uint) (*model.ScheduleActivity, error)
	FindInRange(ctx context.Context, userID uint, from, to time.Time) ([]model.ScheduleActivity, error)
	ReplaceDays(ctx context.Context, activityID uint, days []model.ScheduleActivityDay) error
	AddException(ctx context.Context, activityID uint, date time.Time) error
	SoftDelete(ctx context.Context, userID, id uint) error
}

type scheduleActivityRepo struct {
	db *gorm.DB
	baserepo.BaseRepo[model.ScheduleActivity]
}

func NewScheduleActivityRepo() *scheduleActivityRepo {
	return &scheduleActivityRepo{
		db:       db.DB,
		BaseRepo: baserepo.NewBaseRepo[model.ScheduleActivity](db.DB),
	}
}

func (r *scheduleActivityRepo) WithTx(tx *gorm.DB) ScheduleActivityRepoImpl {
	return &scheduleActivityRepo{
		db:       tx,
		BaseRepo: baserepo.NewBaseRepo[model.ScheduleActivity](tx),
	}
}

// FindOneByID returns the user's activity, with its recurrence days and exceptions.
func (r *scheduleActivityRepo) FindOneByID(ctx context.Context, userID, id uint) (*model.ScheduleActivity, error) {
	var activity model.ScheduleActivity
	if err := r.db.WithContext(ctx).Preload("Days").Preload("Exceptions").
		Where("id = ? AND user_id = ?", id, userID).First(&activity).Error; err != nil {
		return nil, err
	}

	return &activity, nil
}

// FindInRange returns every activity of userID that can have an occurrence in [from, to]: a single
// activity dated inside the range, or a series that started by to and has not ended before from.
// Which exact days inside that window match is decided later by expanding the recurrence.
func (r *scheduleActivityRepo) FindInRange(ctx context.Context, userID uint, from, to time.Time) ([]model.ScheduleActivity, error) {
	activities := []model.ScheduleActivity{}
	if err := r.db.WithContext(ctx).Preload("Days").Preload("Exceptions").
		Where("user_id = ?", userID).
		Where(
			"(recurrence = ? AND date BETWEEN ? AND ?) OR (recurrence <> ? AND date <= ? AND (recurrence_until IS NULL OR recurrence_until >= ?))",
			enums.RecurrenceKindNone, from, to,
			enums.RecurrenceKindNone, to, from,
		).
		Order("date ASC, start_time ASC").
		Find(&activities).Error; err != nil {
		return nil, err
	}

	return activities, nil
}

// ReplaceDays deletes the activity's weekly recurrence days and inserts days instead.
func (r *scheduleActivityRepo) ReplaceDays(ctx context.Context, activityID uint, days []model.ScheduleActivityDay) error {
	if err := r.db.WithContext(ctx).Where("activity_id = ?", activityID).
		Delete(&model.ScheduleActivityDay{}).Error; err != nil {
		return err
	}
	if len(days) == 0 {
		return nil
	}
	for i := range days {
		days[i].ActivityID = activityID
	}
	return r.db.WithContext(ctx).Create(&days).Error
}

// AddException removes one date from the series ("this occurrence only"); a no-op if the date is
// already an exception.
func (r *scheduleActivityRepo) AddException(ctx context.Context, activityID uint, date time.Time) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).
		Create(&model.ScheduleActivityException{ActivityID: activityID, Date: date}).Error
}

// SoftDelete marks the user's activity, and its recurrence days / exceptions, as deleted.
func (r *scheduleActivityRepo) SoftDelete(ctx context.Context, userID, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("id = ? AND user_id = ?", id, userID).Delete(&model.ScheduleActivity{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		if err := tx.Where("activity_id = ?", id).Delete(&model.ScheduleActivityDay{}).Error; err != nil {
			return err
		}
		return tx.Where("activity_id = ?", id).Delete(&model.ScheduleActivityException{}).Error
	})
}
