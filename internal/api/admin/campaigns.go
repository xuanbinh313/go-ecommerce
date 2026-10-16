package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/example/medusa-go-clone/internal/coreflows/promotion/workflows"
	"github.com/example/medusa-go-clone/internal/modules/promotion/domain/models"
	"github.com/gin-gonic/gin"
)

func campaignPage(c *gin.Context) (int, int) {
	limit, offset := 50, 0
	if n, e := strconv.Atoi(c.DefaultQuery("limit", "50")); e == nil && n > 0 {
		limit = n
	}
	if n, e := strconv.Atoi(c.DefaultQuery("offset", "0")); e == nil && n >= 0 {
		offset = n
	}
	return limit, offset
}
func (h *PromotionHTTP) listCampaigns(c *gin.Context) {
	limit, offset := campaignPage(c)
	rows, count, err := h.Service.ListCampaigns(c, models.CampaignFilter{Query: c.Query("q"), Identifier: c.Query("campaign_identifier"), BudgetCurrency: c.Query("budget.currency_code"), Limit: limit, Offset: offset})
	if err != nil {
		writeModuleError(c, err)
		return
	}
	selected := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		selected = append(selected, projectCampaign(row, c.Query("fields")))
	}
	c.JSON(http.StatusOK, gin.H{"campaigns": selected, "count": count, "offset": offset, "limit": limit})
}
func projectCampaign(row models.Campaign, fields string) map[string]any {
	fields = strings.ReplaceAll(fields, "*budget", "budget")
	return projectPaymentValue(row, fields, []string{"id", "name", "description", "currency", "campaign_identifier", "budget", "starts_at", "ends_at", "created_at", "updated_at", "deleted_at"})
}
func (h *PromotionHTTP) getCampaign(c *gin.Context) {
	row, err := h.Service.GetCampaign(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"campaign": projectCampaign(row, c.Query("fields"))})
}
func (h *PromotionHTTP) createCampaign(c *gin.Context) {
	var input models.CampaignInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	row, err := workflows.CreateCampaign(c, h.Service, input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"campaign": projectCampaign(row, c.Query("fields"))})
}
func (h *PromotionHTTP) updateCampaign(c *gin.Context) {
	var input models.CampaignUpdate
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	row, err := workflows.UpdateCampaign(c, h.Service, c.Param("id"), input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"campaign": projectCampaign(row, c.Query("fields"))})
}
func (h *PromotionHTTP) deleteCampaign(c *gin.Context) {
	id := c.Param("id")
	if err := workflows.DeleteCampaign(c, h.Service, id); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "object": "campaign", "deleted": true})
}
func (h *PromotionHTTP) manageCampaignPromotions(c *gin.Context) {
	var input struct {
		Add    []string `json:"add"`
		Remove []string `json:"remove"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if err := workflows.AddOrRemoveCampaignPromotions(c, h.Service, c.Param("id"), input.Add, input.Remove); err != nil {
		writeModuleError(c, err)
		return
	}
	row, err := h.Service.GetCampaign(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"campaign": projectCampaign(row, c.Query("fields"))})
}
