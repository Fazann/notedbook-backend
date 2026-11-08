package repo

import (
	"context"
	"time"

	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/pkg/enums"
	"github.com/Fazann/notedbook-backend/shared/baserepo"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

type BoardSummary struct {
	ID        uint
	Name      string
	CardCount int64
}

type DueCard struct {
	ID          uint
	ColumnID    uint
	Title       string
	Description string
	DueDate     time.Time
	Label       *enums.BoardLabelEnum
	Position    float64
	BoardID     uint
	BoardName   string
	ColumnName  string
}

type BoardRepoImpl interface {
	baserepo.BaseRepo[model.Board]
	ListWithCardCount(ctx context.Context, userID uint) ([]BoardSummary, error)
	FindDueCards(ctx context.Context, userID uint, until time.Time) ([]DueCard, error)

	// Transaction runs fn with a repo bound to one database transaction.
	Transaction(ctx context.Context, fn func(r BoardRepoImpl) error) error

	// FindOwnBoard returns the user's board (gorm.ErrRecordNotFound for another user's).
	FindOwnBoard(ctx context.Context, userID, id uint) (*model.Board, error)
	// FindBoardDetail returns the user's board with its columns and cards, each sorted by position.
	FindBoardDetail(ctx context.Context, userID, id uint) (*model.Board, error)
	CreateBoard(ctx context.Context, board *model.Board) error
	UpdateBoardName(ctx context.Context, id uint, name string) error
	// DeleteBoard soft-deletes the board with its columns and their cards.
	DeleteBoard(ctx context.Context, id uint) error

	// FindOwnColumn returns a column of one of the user's boards.
	FindOwnColumn(ctx context.Context, userID, id uint) (*model.BoardColumn, error)
	// LastColumnPosition is the highest column position on the board, or 0 when it has none.
	LastColumnPosition(ctx context.Context, boardID uint) (float64, error)
	CreateColumn(ctx context.Context, column *model.BoardColumn) error
	UpdateColumnName(ctx context.Context, id uint, name string) error
	// DeleteColumn soft-deletes the column with its cards.
	DeleteColumn(ctx context.Context, id uint) error

	// FindOwnCard returns a card on one of the user's boards, and the board it is on.
	FindOwnCard(ctx context.Context, userID, id uint) (*model.BoardCard, uint, error)
	FindCard(ctx context.Context, id uint) (*model.BoardCard, error)
	// LastCardPosition is the highest card position in the column, or 0 when it is empty.
	LastCardPosition(ctx context.Context, columnID uint) (float64, error)
	CreateCard(ctx context.Context, card *model.BoardCard) error
	UpdateCard(ctx context.Context, id uint, updates map[string]any) error
	DeleteCard(ctx context.Context, id uint) error
}

type boardRepo struct {
	db *gorm.DB
	baserepo.BaseRepo[model.Board]
}

func NewBoardRepo() *boardRepo {
	return &boardRepo{
		db:       db.DB,
		BaseRepo: baserepo.NewBaseRepo[model.Board](db.DB),
	}
}

// ListWithCardCount returns the user's boards, oldest first, with the number of cards on each.
func (r *boardRepo) ListWithCardCount(ctx context.Context, userID uint) ([]BoardSummary, error) {
	rows := []BoardSummary{}
	if err := r.db.WithContext(ctx).Model(&model.Board{}).
		Select("boards.id, boards.name, COUNT(board_cards.id) AS card_count").
		Joins("LEFT JOIN board_columns ON board_columns.board_id = boards.id AND board_columns.deleted_at IS NULL").
		Joins("LEFT JOIN board_cards ON board_cards.column_id = board_columns.id AND board_cards.deleted_at IS NULL").
		Where("boards.user_id = ?", userID).
		Group("boards.id, boards.name").Order("boards.id ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// FindDueCards returns the user's cards due on or before until (overdue included), from every board,
// skipping each board's last column ("Done"). Sorted by due date, then board, column and card position.
func (r *boardRepo) FindDueCards(ctx context.Context, userID uint, until time.Time) ([]DueCard, error) {
	rows := []DueCard{}
	if err := r.db.WithContext(ctx).Model(&model.BoardCard{}).
		Select("board_cards.id, board_cards.column_id, board_cards.title, board_cards.description, "+
			"board_cards.due_date, board_cards.label, board_cards.position, "+
			"boards.id AS board_id, boards.name AS board_name, board_columns.name AS column_name").
		Joins("JOIN board_columns ON board_columns.id = board_cards.column_id AND board_columns.deleted_at IS NULL").
		Joins("JOIN boards ON boards.id = board_columns.board_id AND boards.deleted_at IS NULL").
		Where("boards.user_id = ? AND board_cards.due_date IS NOT NULL AND board_cards.due_date <= ?", userID, until).
		Where("board_columns.position < (SELECT MAX(last.position) FROM board_columns AS last " +
			"WHERE last.board_id = boards.id AND last.deleted_at IS NULL)").
		Order("board_cards.due_date ASC, boards.id ASC, board_columns.position ASC, board_cards.position ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
