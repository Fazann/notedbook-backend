package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Fazann/notedbook-backend/pkg/i18n"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

type defaultPlanArea struct {
	title        string
	translations map[i18n.Lang]string
	icon         string
}

// defaultPlanAreas are shared by every user. The English title is the base title.
var defaultPlanAreas = []defaultPlanArea{
	{"Personal", map[i18n.Lang]string{i18n.KM: "ផ្ទាល់ខ្លួន", i18n.MS: "Peribadi"}, "user"},
	{"Career", map[i18n.Lang]string{i18n.KM: "ការងារ", i18n.MS: "Kerjaya"}, "briefcase"},
	{"Health", map[i18n.Lang]string{i18n.KM: "សុខភាព", i18n.MS: "Kesihatan"}, "heart-pulse"},
	{"Finance", map[i18n.Lang]string{i18n.KM: "ហិរញ្ញវត្ថុ", i18n.MS: "Kewangan"}, "wallet"},
	{"Learning", map[i18n.Lang]string{i18n.KM: "ការសិក្សា", i18n.MS: "Pembelajaran"}, "graduation-cap"},
	{"Family", map[i18n.Lang]string{i18n.KM: "គ្រួសារ", i18n.MS: "Keluarga"}, "users"},
}

// SeedDefaultPlanAreas creates the default plan areas that do not exist yet and returns how many it created.
// Safe to run again: existing (also deleted) areas with the same title are skipped.
func SeedDefaultPlanAreas(ctx context.Context) (int, error) {
	return NewPlanAreaSvc().seedDefaults(ctx)
}

func (s *planAreaSvc) seedDefaults(ctx context.Context) (created int, err error) {
	defer func() {
		if created > 0 {
			s.invalidateCache(ctx)
		}
	}()

	for _, item := range defaultPlanAreas {
		_, err := s.planAreaRepo.FindByTitle(ctx, item.title)
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return created, fmt.Errorf("find default plan area %q: %w", item.title, err)
		}

		err = s.transaction(ctx, func(tx *gorm.DB) error {
			repoTx := s.planAreaRepo.WithTx(tx)
			area, err := repoTx.Create(ctx, &model.PlanArea{
				Title: item.title,
				Icon:  item.icon,
			})
			if err != nil {
				return err
			}
			return repoTx.ReplaceTranslations(ctx, area.ID, toPlanAreaTranslationModels(item.translations))
		})
		if err != nil {
			return created, fmt.Errorf("seed default plan area %q: %w", item.title, err)
		}
		created++
	}
	return created, nil
}
