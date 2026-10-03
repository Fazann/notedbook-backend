package service

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Fazann/notedbook-backend/internal/plans/dto"
	"github.com/Fazann/notedbook-backend/internal/plans/repo"
	"github.com/Fazann/notedbook-backend/pkg/db"
	"github.com/Fazann/notedbook-backend/pkg/i18n"
	"github.com/Fazann/notedbook-backend/pkg/response"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

type PlanAreaServiceImpl interface {
	List(ctx context.Context, lang i18n.Lang) response.SvcRes
}

type planAreaSvc struct {
	planAreaRepo repo.PlanAreaRepoImpl
	// Runs fn in a database transaction; replaced in tests.
	transaction func(ctx context.Context, fn func(tx *gorm.DB) error) error
}

func NewPlanAreaSvc() *planAreaSvc {
	return &planAreaSvc{
		planAreaRepo: repo.NewPlanAreaRepo(),
		transaction: func(ctx context.Context, fn func(tx *gorm.DB) error) error {
			return db.DB.WithContext(ctx).Transaction(fn)
		},
	}
}

func (s *planAreaSvc) List(ctx context.Context, lang i18n.Lang) response.SvcRes {
	areas, err := s.planAreaRepo.FindAll(ctx)
	if err != nil {
		return response.SvcRespErr(fmt.Errorf("list plan areas: %w", err), http.StatusInternalServerError)
	}

	return response.SvcRespSuccess(toPlanAreaResList(areas, lang), http.StatusOK)
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
