package models

import "time"

type Translation struct {
	ID                   string            `json:"id"`
	ReferenceID          string            `json:"reference_id"`
	Reference            string            `json:"reference"`
	LocaleCode           string            `json:"locale_code"`
	Translations         map[string]string `json:"translations"`
	TranslatedFieldCount int               `json:"translated_field_count"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
}
type Locale struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type LocaleFilter struct {
	Codes []string
	Query string
}
type CreateInput struct {
	ReferenceID  string            `json:"reference_id"`
	Reference    string            `json:"reference"`
	LocaleCode   string            `json:"locale_code"`
	Translations map[string]string `json:"translations"`
}
type UpdateInput struct {
	ID           string            `json:"id"`
	ReferenceID  *string           `json:"reference_id"`
	Reference    *string           `json:"reference"`
	LocaleCode   *string           `json:"locale_code"`
	Translations map[string]string `json:"translations"`
}
type BatchInput struct {
	Create []CreateInput `json:"create"`
	Update []UpdateInput `json:"update"`
	Delete []string      `json:"delete"`
}
type Filter struct {
	ReferenceIDs, References, LocaleCodes []string
	Limit, Offset                         int
}

type TranslationSetting struct {
	ID         string    `json:"id"`
	EntityType string    `json:"entity_type"`
	Fields     []string  `json:"fields"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type TranslationSettingsFilter struct {
	IDs        []string
	EntityType *string
	IsActive   *bool
}

type CreateTranslationSettingInput struct {
	EntityType string   `json:"entity_type"`
	Fields     []string `json:"fields"`
	IsActive   *bool    `json:"is_active"`
}

type UpdateTranslationSettingInput struct {
	ID         string    `json:"id"`
	EntityType *string   `json:"entity_type"`
	Fields     *[]string `json:"fields"`
	IsActive   *bool     `json:"is_active"`
}

type BatchTranslationSettingsInput struct {
	Create []CreateTranslationSettingInput `json:"create"`
	Update []UpdateTranslationSettingInput `json:"update"`
	Delete []string                        `json:"delete"`
}
