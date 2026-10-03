package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Fazann/notedbook-backend/internal/plans/dto"
	"github.com/Fazann/notedbook-backend/internal/plans/repo"
	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/logger"
	"github.com/Fazann/notedbook-backend/pkg/redis"
	"github.com/Fazann/notedbook-backend/pkg/response"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

const (
	// Holds every plan area with its translations as JSON; the same for all users and languages.
	planAreaCacheKey = "plan_areas:all"
	planAreaCacheTTL = 24 * time.Hour
)

type PlanAreaServiceImpl interface {
	List(ctx context.Context, lang i18n.Lang) response.SvcRes
	ResetCache(ctx context.Context) response.SvcRes
}

type planAreaCache interface {
	Get(ctx context.Context, key string) (string, bool, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Del(ctx context.Context, keys ...string) error
}

type planAreaSvc struct {
	planAreaRepo repo.PlanAreaRepoImpl
	cache        planAreaCache
	// Runs fn in a database transaction; replaced in tests.
	transaction func(ctx context.Context, fn func(tx *gorm.DB) error) error
}

func NewPlanAreaSvc() *planAreaSvc {
	return &planAreaSvc{
		planAreaRepo: repo.NewPlanAreaRepo(),
		cache:        redis.Store{},
		transaction: func(ctx context.Context, fn func(tx *gorm.DB) error) error {
			return db.DB.WithContext(ctx).Transaction(fn)
		},
	}
}

func (s *planAreaSvc) List(ctx context.Context, lang i18n.Lang) response.SvcRes {
	areas, err := s.findAll(ctx)
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("list plan areas: %w", err), http.StatusInternalServerError)
	}

	return response.SvcRespSuccess(toPlanAreaResList(areas, lang), http.StatusOK)
}

func (s *planAreaSvc) ResetCache(ctx context.Context) response.SvcRes {
	if err := s.cache.Del(ctx, planAreaCacheKey); err != nil {
		return response.SvcRespErr(fmt.Errorf("reset plan area cache: %w", err), http.StatusInternalServerError)
	}

	return response.SvcRespSuccess(nil, http.StatusOK)
}

// findAll reads the plan areas from the cache, or from the database and then caches them.
// A cache failure is logged and never fails the request.
func (s *planAreaSvc) findAll(ctx context.Context) ([]model.PlanArea, error) {
	cached, found, err := s.cache.Get(ctx, planAreaCacheKey)
	if err != nil {
		logger.Warn("read plan area cache failed", "err", err)
	}
	if found {
		var areas []model.PlanArea
		if err := json.Unmarshal([]byte(cached), &areas); err == nil {
			return areas, nil
		}
		logger.Warn("decode plan area cache failed", "err", err)
	}

	areas, err := s.planAreaRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	if data, err := json.Marshal(areas); err != nil {
		logger.Warn("encode plan area cache failed", "err", err)
	} else if err := s.cache.Set(ctx, planAreaCacheKey, string(data), planAreaCacheTTL); err != nil {
		logger.Warn("write plan area cache failed", "err", err)
	}
	return areas, nil
}

// invalidateCache drops the cached plan areas; call it after every create, update or delete of a plan area.
// A failure is only logged: the cache still expires after planAreaCacheTTL.
func (s *planAreaSvc) invalidateCache(ctx context.Context) {
	if err := s.cache.Del(ctx, planAreaCacheKey); err != nil {
		logger.Warn("invalidate plan area cache failed", "err", err)
	}
}

func toPlanAreaTranslationModels(translations map[i18n.Lang]string) []model.PlanAreaTr {
	trs := make([]model.PlanAreaTr, 0, len(translations))
	for _, lang := range i18n.Langs {
		if title, ok := translations[lang]; ok {
			trs = append(trs, model.PlanAreaTr{Lang: string(lang), Title: title})
		}
	}
	return trs
}

func toPlanAreaRes(area *model.PlanArea, lang i18n.Lang) *dto.PlanAreaRes {
	res := &dto.PlanAreaRes{
		ID:           area.ID,
		Title:        area.Title,
		DisplayTitle: area.Title,
		Translations: make(map[string]string, len(area.Translations)),
		Icon:         area.Icon,
		CreatedAt:    area.CreatedAt,
		UpdatedAt:    area.UpdatedAt,
	}
	for _, tr := range area.Translations {
		res.Translations[tr.Lang] = tr.Title
		if tr.Lang == string(lang) && tr.Title != "" {
			res.DisplayTitle = tr.Title
		}
	}
	return res
}

func toPlanAreaResList(areas []model.PlanArea, lang i18n.Lang) []*dto.PlanAreaRes {
	res := make([]*dto.PlanAreaRes, len(areas))
	for i := range areas {
		res[i] = toPlanAreaRes(&areas[i], lang)
	}
	return res
}
