package admin

import (
	"context"
	"github.com/example/medusa-go-clone/internal/coreflows/translation/workflows"
	"github.com/example/medusa-go-clone/internal/modules/translation/domain/models"
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
)

type TranslationService interface {
	ListLocales(context.Context, models.LocaleFilter, int, int) ([]models.Locale, int, error)
	GetLocaleByCode(context.Context, string) (models.Locale, error)
	List(context.Context, models.Filter) ([]models.Translation, int, error)
	Batch(context.Context, models.BatchInput) ([]models.Translation, []models.Translation, error)
	ListSettings(context.Context, models.TranslationSettingsFilter) ([]models.TranslationSetting, error)
	CreateSettings(context.Context, []models.CreateTranslationSettingInput) ([]models.TranslationSetting, error)
	UpdateSettings(context.Context, []models.UpdateTranslationSettingInput) ([]models.TranslationSetting, error)
	DeleteSettings(context.Context, []string) error
	RestoreSettings(context.Context, []models.TranslationSetting) error
}
type TranslationHTTP struct {
	Service     TranslationService
	Enabled     bool
	Permissions UserRolesWorkflow
	RBACEnabled bool
}

func (h *TranslationHTTP) Register(router *gin.RouterGroup, auth gin.HandlerFunc) {
	if !h.Enabled {
		return
	}
	a := router.Group("/admin/translations", auth)
	a.GET("", permissionMiddleware(h.Permissions, h.RBACEnabled, "translation", "read"), h.list)
	a.POST("/batch", permissionMiddleware(h.Permissions, h.RBACEnabled, "translation", "read", "translation", "create", "translation", "update"), h.batch)
	a.GET("/settings", permissionMiddleware(h.Permissions, h.RBACEnabled, "translation", "read", "translation_setting", "read"), h.listSettings)
	a.POST("/settings/batch", permissionMiddleware(h.Permissions, h.RBACEnabled, "translation", "read", "translation_setting", "create", "translation_setting", "update"), h.batchSettings)
	locales := router.Group("/admin", auth)
	locales.GET("/locales", permissionMiddleware(h.Permissions, h.RBACEnabled, "store_locale", "read"), h.listLocales)
	locales.GET("/locales/:code", permissionMiddleware(h.Permissions, h.RBACEnabled, "store_locale", "read"), h.getLocale)
}

func localeFields(locale models.Locale, requested string) map[string]any {
	fields := []string{"code", "name"}
	if requested != "" {
		fields = strings.Split(requested, ",")
	}
	out := map[string]any{}
	for _, field := range fields {
		switch strings.TrimSpace(field) {
		case "*", "code":
			out["code"] = locale.Code
		case "name":
			out["name"] = locale.Name
		}
	}
	return out
}
func (h *TranslationHTTP) listLocales(c *gin.Context) {
	limit, offset := 200, 0
	if n, e := strconv.Atoi(c.DefaultQuery("limit", "200")); e == nil && n > 0 {
		limit = n
	}
	if n, e := strconv.Atoi(c.DefaultQuery("offset", "0")); e == nil && n >= 0 {
		offset = n
	}
	rows, count, err := h.Service.ListLocales(c, models.LocaleFilter{Codes: queryValues(c, "code"), Query: c.Query("q")}, limit, offset)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	items := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		items = append(items, localeFields(row, c.Query("fields")))
	}
	c.JSON(200, gin.H{"locales": items, "count": count, "offset": offset, "limit": limit})
}
func (h *TranslationHTTP) getLocale(c *gin.Context) {
	row, err := h.Service.GetLocaleByCode(c, c.Param("code"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"locale": localeFields(row, c.Query("fields"))})
}
func (h *TranslationHTTP) list(c *gin.Context) {
	limit, offset := 20, 0
	if n, e := strconv.Atoi(c.DefaultQuery("limit", "20")); e == nil && n > 0 {
		limit = n
	}
	if n, e := strconv.Atoi(c.DefaultQuery("offset", "0")); e == nil && n >= 0 {
		offset = n
	}
	f := models.Filter{ReferenceIDs: queryValues(c, "reference_id"), References: queryValues(c, "reference"), LocaleCodes: queryValues(c, "locale_code"), Limit: limit, Offset: offset}
	rows, count, err := h.Service.List(c, f)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, listResponse("translations", rows, count, offset, limit))
}
func (h *TranslationHTTP) batch(c *gin.Context) {
	var in models.BatchInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	created, updated, err := workflows.BatchTranslations(c, h.Service, in)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"created": created, "updated": updated, "deleted": gin.H{"ids": in.Delete, "object": "translation", "deleted": true}})
}

func (h *TranslationHTTP) listSettings(c *gin.Context) {
	var entityType *string
	if value := c.Query("entity_type"); value != "" {
		entityType = &value
	}
	var isActive *bool
	if value := c.Query("is_active"); value != "" {
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			c.JSON(400, gin.H{"type": "invalid_data", "message": "is_active must be a boolean"})
			return
		}
		isActive = &parsed
	}
	settings, err := h.Service.ListSettings(c, models.TranslationSettingsFilter{EntityType: entityType, IsActive: isActive})
	if err != nil {
		writeModuleError(c, err)
		return
	}
	result := map[string]any{}
	for _, setting := range settings {
		result[setting.EntityType] = gin.H{"id": setting.ID, "fields": setting.Fields, "inactive_fields": []string{}, "is_active": setting.IsActive}
	}
	c.JSON(200, gin.H{"translation_settings": result})
}

func (h *TranslationHTTP) batchSettings(c *gin.Context) {
	var input models.BatchTranslationSettingsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	for _, setting := range input.Create {
		if setting.EntityType == "" || setting.Fields == nil {
			c.JSON(400, gin.H{"type": "invalid_data", "message": "entity_type and fields are required"})
			return
		}
	}
	created, updated, err := workflows.BatchTranslationSettings(c, h.Service, input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(200, gin.H{"created": created, "updated": updated, "deleted": gin.H{"ids": input.Delete, "object": "translation_settings", "deleted": true}})
}
