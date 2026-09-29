package dto

import "time"

type CategoryReq struct {
	// Base name, shown when there is no translation for the request language.
	Name string `json:"name" binding:"required,max=50" example:"Gym"`
	// Optional name per language (en, km, ms), e.g. {"km": "ហាត់ប្រាណ"}. A blank value is dropped.
	Translations map[string]string `json:"translations" binding:"omitempty,max=3"`
	Icon         string            `json:"icon" binding:"required" example:"dumbbell"`
	Color        string            `json:"color" binding:"required" example:"green"`
}

// CategoryUpdateReq changes only the fields sent; a field left out (or null) keeps its value.
type CategoryUpdateReq struct {
	Name *string `json:"name" binding:"omitempty,max=50" example:"Gym"`
	// When sent, replaces all translations: languages left out or blank are removed, {} removes them all.
	Translations map[string]string `json:"translations" binding:"omitempty,max=3"`
	Icon         *string           `json:"icon" example:"dumbbell"`
	Color        *string           `json:"color" example:"green"`
}

type CategoryListReq struct {
	Page  int `form:"page" binding:"omitempty,min=1" example:"1"`
	Limit int `form:"limit" binding:"omitempty,min=1,max=1000" example:"10"`
	// Matches the base name and every translation, in any language.
	Q    string `form:"q" binding:"omitempty,max=100" example:"gym"`
	Sort string `form:"sort" binding:"omitempty,oneof=name -name created_at -created_at" example:"name"`
}

type CategoryRes struct {
	ID   uint   `json:"id" example:"1"`
	Name string `json:"name" example:"Gym"`
	// Name in the request language (`?lang=` / Accept-Language), falling back to name.
	DisplayName  string            `json:"display_name" example:"ហាត់ប្រាណ"`
	Translations map[string]string `json:"translations"`
	// Created by an admin for every user; read-only (cannot be updated or deleted by the user).
	IsDefault bool      `json:"is_default" example:"false"`
	Icon      string    `json:"icon" example:"dumbbell"`
	Color     string    `json:"color" example:"green"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
