package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Fazann/notedbook-backend/pkg/i18n"
	model "github.com/Fazann/notedbook-backend/shared/models"
	"gorm.io/gorm"
)

type defaultCategory struct {
	name         string
	translations map[i18n.Lang]string
	icon         string
	color        string
}

// defaultCategories are shared by every user (user_id NULL). The English name is the base name.
var defaultCategories = []defaultCategory{
	{"Food", map[i18n.Lang]string{i18n.KM: "អាហារ", i18n.MS: "Makanan"}, "utensils", "chart-1"},
	{"Coffee", map[i18n.Lang]string{i18n.KM: "កាហ្វេ", i18n.MS: "Kopi"}, "coffee", "amber"},
	{"Transport", map[i18n.Lang]string{i18n.KM: "ការធ្វើដំណើរ", i18n.MS: "Pengangkutan"}, "bike", "blue"},
	{"Rent", map[i18n.Lang]string{i18n.KM: "ថ្លៃផ្ទះ", i18n.MS: "Sewa"}, "home", "violet"},
	{"Phone & Internet", map[i18n.Lang]string{i18n.KM: "ទូរស័ព្ទ និងអ៊ីនធឺណិត", i18n.MS: "Telefon & Internet"}, "wifi", "teal"},
	{"Family", map[i18n.Lang]string{i18n.KM: "គ្រួសារ", i18n.MS: "Keluarga"}, "users", "pink"},
	{"Shopping", map[i18n.Lang]string{i18n.KM: "ការទិញទំនិញ", i18n.MS: "Beli-belah"}, "shopping-bag", "orange"},
	{"Health", map[i18n.Lang]string{i18n.KM: "សុខភាព", i18n.MS: "Kesihatan"}, "heart-pulse", "red"},
	{"Education", map[i18n.Lang]string{i18n.KM: "ការសិក្សា", i18n.MS: "Pendidikan"}, "book-open", "chart-2"},
	{"Other", map[i18n.Lang]string{i18n.KM: "ផ្សេងៗ", i18n.MS: "Lain-lain"}, "ellipsis", "slate"},
}

// SeedDefaultCategories creates the default categories that do not exist yet and returns how many it created.
// Safe to run again: existing (also deleted) defaults with the same name are skipped.
func SeedDefaultCategories(ctx context.Context) (int, error) {
	return NewCategorySvc().seedDefaults(ctx)
}

func (s *categorySvc) seedDefaults(ctx context.Context) (int, error) {
	created := 0
	for _, item := range defaultCategories {
		_, err := s.categoryRepo.FindDefaultByName(ctx, item.name)
		if err == nil {
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return created, fmt.Errorf("find default category %q: %w", item.name, err)
		}

		err = s.transaction(ctx, func(tx *gorm.DB) error {
			repoTx := s.categoryRepo.WithTx(tx)
			category, err := repoTx.Create(ctx, &model.ExpenseCategory{
				Name:       item.name,
				NameSearch: buildNameSearch(item.name, item.translations),
				Icon:       item.icon,
				Color:      item.color,
			})
			if err != nil {
				return err
			}
			return repoTx.ReplaceTranslations(ctx, category.ID, toTranslationModels(item.translations))
		})
		if err != nil {
			return created, fmt.Errorf("seed default category %q: %w", item.name, err)
		}
		created++
	}
	return created, nil
}
