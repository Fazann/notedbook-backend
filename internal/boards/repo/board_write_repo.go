package repo

import (
	"context"

	"github.com/Fazann/notedbook-backend/shared/baserepo"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

func (r *boardRepo) Transaction(ctx context.Context, fn func(r BoardRepoImpl) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&boardRepo{db: tx, BaseRepo: baserepo.NewBaseRepo[model.Board](tx)})
	})
}

func (r *boardRepo) FindOwnBoard(ctx context.Context, userID, id uint) (*model.Board, error) {
	var board model.Board
	if err := r.db.WithContext(ctx).Where("id = ? AND user_id = ?", id, userID).First(&board).Error; err != nil {
		return nil, err
	}
	return &board, nil
}

func (r *boardRepo) FindBoardDetail(ctx context.Context, userID, id uint) (*model.Board, error) {
	var board model.Board
	err := r.db.WithContext(ctx).
		Preload("Columns", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC, id ASC") }).
		Preload("Columns.Cards", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC, id ASC") }).
		Where("id = ? AND user_id = ?", id, userID).First(&board).Error
	if err != nil {
		return nil, err
	}
	return &board, nil
}

func (r *boardRepo) CreateBoard(ctx context.Context, board *model.Board) error {
	return r.db.WithContext(ctx).Create(board).Error
}

func (r *boardRepo) UpdateBoardName(ctx context.Context, id uint, name string) error {
	return r.db.WithContext(ctx).Model(&model.Board{}).Where("id = ?", id).Update("name", name).Error
}

func (r *boardRepo) DeleteBoard(ctx context.Context, id uint) error {
	db := r.db.WithContext(ctx)
	columnIDs := db.Model(&model.BoardColumn{}).Select("id").Where("board_id = ?", id)
	if err := db.Where("column_id IN (?)", columnIDs).Delete(&model.BoardCard{}).Error; err != nil {
		return err
	}
	if err := db.Where("board_id = ?", id).Delete(&model.BoardColumn{}).Error; err != nil {
		return err
	}
	return db.Where("id = ?", id).Delete(&model.Board{}).Error
}

func (r *boardRepo) FindOwnColumn(ctx context.Context, userID, id uint) (*model.BoardColumn, error) {
	var column model.BoardColumn
	err := r.db.WithContext(ctx).
		Joins("JOIN boards ON boards.id = board_columns.board_id AND boards.deleted_at IS NULL").
		Where("board_columns.id = ? AND boards.user_id = ?", id, userID).
		First(&column).Error
	if err != nil {
		return nil, err
	}
	return &column, nil
}

func (r *boardRepo) LastColumnPosition(ctx context.Context, boardID uint) (float64, error) {
	var last float64
	err := r.db.WithContext(ctx).Model(&model.BoardColumn{}).Where("board_id = ?", boardID).
		Select("COALESCE(MAX(position), 0)").Scan(&last).Error
	return last, err
}

func (r *boardRepo) CreateColumn(ctx context.Context, column *model.BoardColumn) error {
	return r.db.WithContext(ctx).Create(column).Error
}

func (r *boardRepo) UpdateColumnName(ctx context.Context, id uint, name string) error {
	return r.db.WithContext(ctx).Model(&model.BoardColumn{}).Where("id = ?", id).Update("name", name).Error
}

func (r *boardRepo) DeleteColumn(ctx context.Context, id uint) error {
	db := r.db.WithContext(ctx)
	if err := db.Where("column_id = ?", id).Delete(&model.BoardCard{}).Error; err != nil {
		return err
	}
	return db.Where("id = ?", id).Delete(&model.BoardColumn{}).Error
}

func (r *boardRepo) FindOwnCard(ctx context.Context, userID, id uint) (*model.BoardCard, uint, error) {
	var row struct {
		model.BoardCard
		BoardID uint
	}
	err := r.db.WithContext(ctx).Model(&model.BoardCard{}).
		Select("board_cards.*, board_columns.board_id").
		Joins("JOIN board_columns ON board_columns.id = board_cards.column_id AND board_columns.deleted_at IS NULL").
		Joins("JOIN boards ON boards.id = board_columns.board_id AND boards.deleted_at IS NULL").
		Where("board_cards.id = ? AND boards.user_id = ?", id, userID).
		Take(&row).Error
	if err != nil {
		return nil, 0, err
	}
	return &row.BoardCard, row.BoardID, nil
}

func (r *boardRepo) FindCard(ctx context.Context, id uint) (*model.BoardCard, error) {
	var card model.BoardCard
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&card).Error; err != nil {
		return nil, err
	}
	return &card, nil
}

func (r *boardRepo) LastCardPosition(ctx context.Context, columnID uint) (float64, error) {
	var last float64
	err := r.db.WithContext(ctx).Model(&model.BoardCard{}).Where("column_id = ?", columnID).
		Select("COALESCE(MAX(position), 0)").Scan(&last).Error
	return last, err
}

func (r *boardRepo) CreateCard(ctx context.Context, card *model.BoardCard) error {
	return r.db.WithContext(ctx).Create(card).Error
}

func (r *boardRepo) UpdateCard(ctx context.Context, id uint, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&model.BoardCard{}).Where("id = ?", id).Updates(updates).Error
}

func (r *boardRepo) DeleteCard(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.BoardCard{}).Error
}
