package baserepo

import (
	"context"
	"errors"

	"github.com/Fazann/notedbook-backend/pkg/constants"
	"gorm.io/gorm"
)

// Noted: make sure T is not pointer type
type BaseRepo[T any] interface {
	Create(ctx context.Context, data *T) (*T, error)
	Save(ctx context.Context, data *T) error
	FindOne(ctx context.Context, customCond ...any) (*T, error)
	Delete(ctx context.Context, cond ...any) error
	HardDelete(ctx context.Context, cond ...any) error
	Update(ctx context.Context, condition any, toUpdatePayload map[string]any) error
	UpdateWithModel(ctx context.Context, condition T, toUpdatePayload *T) error
}

type baseRepo[T any] struct {
	db *gorm.DB
}

func NewBaseRepo[T any](db *gorm.DB) BaseRepo[T] {
	return &baseRepo[T]{
		db: db,
	}
}

func (r *baseRepo[T]) Create(ctx context.Context, data *T) (*T, error) {
	res := r.db.WithContext(ctx).Create(data)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, errors.New(constants.NoRowAffectedMsg)
	}
	return data, nil
}

func (r *baseRepo[T]) Save(ctx context.Context, data *T) error {
	res := r.db.WithContext(ctx).Save(data)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New(constants.NoRowAffectedMsg)
	}
	return nil
}

// if there's no customCond string it will automatically search primary key
func (r *baseRepo[T]) FindOne(ctx context.Context, customCond ...any) (*T, error) {
	var data T
	if err := r.db.WithContext(ctx).First(&data, customCond...).Error; err != nil {
		return nil, err
	}
	return &data, nil
}

func (r *baseRepo[T]) Delete(ctx context.Context, cond ...any) error {
	res := r.db.WithContext(ctx).Delete(new(*T), cond...)
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (r *baseRepo[T]) HardDelete(ctx context.Context, cond ...any) error {
	res := r.db.WithContext(ctx).Unscoped().Delete(new(T), cond...)
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (r *baseRepo[T]) Update(ctx context.Context, condition any, toUpdatePayload map[string]any) error {
	res := r.db.WithContext(ctx).Model(new(T)).Where(condition).Updates(toUpdatePayload)
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (r *baseRepo[T]) UpdateWithModel(ctx context.Context, condition T, toUpdatePayload *T) error {
	res := r.db.WithContext(ctx).Model(new(T)).Where(condition).Updates(toUpdatePayload)
	if res.Error != nil {
		return res.Error
	}
	return nil
}
