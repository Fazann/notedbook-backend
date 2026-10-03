package repo

import (
	"context"
	"time"

	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/shared/baserepo"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

// progressExpr is a plan's progress (0-100) in SQL, the same rule as the API response: 100 once DONE,
// 0 without steps, else the rounded share of done steps.
const progressExpr = "CASE WHEN status = 'DONE' THEN 100 WHEN step_count = 0 THEN 0 " +
	"ELSE ROUND(done_step_count * 100.0 / step_count) END"

// PlanFilter is the plan list query. Zero filters and nil sort fields are not applied.
type PlanFilter struct {
	Statuses []string `filter:"column=status;op=in"`
	AreaIDs  []int    `filter:"column=area_id;op=in"`
	// Lowercased, with LIKE wildcards escaped.
	Search string `filter:"column=LOWER(title);op=like"`
	// "target_date IS NULL" first keeps plans without a date last in both directions.
	SortTargetDate *string `sort:"target_date IS NULL, target_date"`
	SortCreatedAt  *string `sort:"created_at"`
	SortTitle      *string `sort:"LOWER(title)"`
	// Same rule as progressExpr (struct tags must be literals).
	SortProgress *string `sort:"CASE WHEN status = 'DONE' THEN 100 WHEN step_count = 0 THEN 0 ELSE ROUND(done_step_count * 100.0 / step_count) END"`
	// Tie-breaker, so pages do not overlap when the sort value repeats.
	SortID *string `sort:"id"`
}

type PlanRepoImpl interface {
	baserepo.BaseRepo[model.Plan]
	WithTx(tx *gorm.DB) PlanRepoImpl
	FindOneByID(ctx context.Context, userID, id uint) (*model.Plan, error)
	List(ctx context.Context, userID uint, filter *PlanFilter, page, limit int) ([]model.Plan, int64, error)
	// AdjustStepCounts atomically changes step_count and done_step_count by the given deltas.
	AdjustStepCounts(ctx context.Context, planID uint, stepDelta, doneDelta int) error
	SoftDelete(ctx context.Context, userID, id uint) error
	// Stats counts the user's plans; today decides "overdue", [yearStart, nextYear) decides "done this year".
	Stats(ctx context.Context, userID uint, today, yearStart, nextYear time.Time) (*PlanStats, error)
}

type PlanStats struct {
	Active          int
	DoneThisYear    int
	Overdue         int
	AverageProgress float64
}

type planRepo struct {
	db *gorm.DB
	baserepo.BaseRepo[model.Plan]
}

func NewPlanRepo() *planRepo {
	return &planRepo{
		db:       db.DB,
		BaseRepo: baserepo.NewBaseRepo[model.Plan](db.DB),
	}
}

func (r *planRepo) WithTx(tx *gorm.DB) PlanRepoImpl {
	return &planRepo{
		db:       tx,
		BaseRepo: baserepo.NewBaseRepo[model.Plan](tx),
	}
}

// FindOneByID returns the user's plan.
func (r *planRepo) FindOneByID(ctx context.Context, userID, id uint) (*model.Plan, error) {
	var plan model.Plan
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&plan).Error; err != nil {
		return nil, err
	}

	return &plan, nil
}

func (r *planRepo) List(ctx context.Context, userID uint, filter *PlanFilter, page, limit int) ([]model.Plan, int64, error) {
	return baserepo.NewQueryBuilder[model.Plan](r.db.Where("user_id = ?", userID)).
		ApplyFilters(filter).
		Paginate(ctx, page, limit)
}

func (r *planRepo) AdjustStepCounts(ctx context.Context, planID uint, stepDelta, doneDelta int) error {
	return r.db.WithContext(ctx).Model(&model.Plan{}).Where("id = ?", planID).
		Updates(map[string]any{
			"step_count":      gorm.Expr("step_count + ?", stepDelta),
			"done_step_count": gorm.Expr("done_step_count + ?", doneDelta),
		}).Error
}

// SoftDelete marks the user's plan, and its steps, as deleted.
func (r *planRepo) SoftDelete(ctx context.Context, userID, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Plan{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return tx.Where("plan_id = ?", id).Delete(&model.PlanStep{}).Error
	})
}

func (r *planRepo) Stats(ctx context.Context, userID uint, today, yearStart, nextYear time.Time) (*PlanStats, error) {
	var stats PlanStats
	err := r.db.WithContext(ctx).Model(&model.Plan{}).
		Select(
			"COUNT(*) FILTER (WHERE status <> 'DONE') AS active, "+
				"COUNT(*) FILTER (WHERE status = 'DONE' AND done_at >= ? AND done_at < ?) AS done_this_year, "+
				"COUNT(*) FILTER (WHERE status <> 'DONE' AND target_date < ?) AS overdue, "+
				"COALESCE(AVG("+progressExpr+") FILTER (WHERE status <> 'DONE'), 0) AS average_progress",
			yearStart, nextYear, today,
		).
		Where("user_id = ?", userID).
		Scan(&stats).Error
	if err != nil {
		return nil, err
	}
	return &stats, nil
}
