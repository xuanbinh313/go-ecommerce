package admin

import (
	"context"
	"net/http"

	userflows "github.com/example/medusa-go-clone/internal/coreflows/user/workflows"
	"github.com/example/medusa-go-clone/internal/modules/user/domain/models"
	"github.com/gin-gonic/gin"
)

type InviteReader interface {
	GetInvite(context.Context, string) (models.Invite, error)
	ListInvitesPage(context.Context, []string, []string, int, int) ([]models.Invite, int, error)
}
type InviteWorkflow interface {
	CreateInvite(context.Context, userflows.CreateInviteInput) (models.Invite, error)
	DeleteInvite(context.Context, string) error
	RefreshInviteToken(context.Context, string) (models.Invite, error)
	AcceptInvite(context.Context, string, string, *string, *string, *string) (models.User, error)
}
type InviteHTTP struct {
	Service  InviteReader
	Workflow InviteWorkflow
}

func (h *InviteHTTP) Register(router *gin.RouterGroup, authenticate gin.HandlerFunc) {
	admin := router.Group("/admin", authenticate)
	admin.GET("/invites", h.list)
	admin.POST("/invites", h.create)
	admin.GET("/invites/:id", h.get)
	admin.DELETE("/invites/:id", h.delete)
	admin.POST("/invites/:id/resend", h.resend)
}
func (h *InviteHTTP) RegisterAccept(router *gin.RouterGroup, authenticate gin.HandlerFunc) {
	router.POST("/admin/invites/accept", authenticate, h.accept)
}
func (h *InviteHTTP) list(c *gin.Context) {
	limit, offset := paginationFor(c, 50, 100)
	rows, count, err := h.Service.ListInvitesPage(c, queryIDs(c), c.QueryArray("email"), limit, offset)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"invites": rows, "count": count, "offset": offset, "limit": limit})
}
func (h *InviteHTTP) get(c *gin.Context) {
	row, err := h.Service.GetInvite(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"invite": row})
}
func (h *InviteHTTP) create(c *gin.Context) {
	var input userflows.CreateInviteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	if input.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "email is required"})
		return
	}
	row, err := h.Workflow.CreateInvite(c, input)
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"invite": row})
}
func (h *InviteHTTP) delete(c *gin.Context) {
	id := c.Param("id")
	if err := h.Workflow.DeleteInvite(c, id); err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "object": "invite", "deleted": true})
}
func (h *InviteHTTP) resend(c *gin.Context) {
	row, err := h.Workflow.RefreshInviteToken(c, c.Param("id"))
	if err != nil {
		writeModuleError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"invite": row})
}

func (h *InviteHTTP) accept(c *gin.Context) {
	if actorID := contextValue(c.GetString("actor_id")); actorID != "" {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": "The user is already authenticated and cannot accept an invite."})
		return
	}
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}
	var input struct {
		Email     *string `json:"email"`
		FirstName *string `json:"first_name"`
		LastName  *string `json:"last_name"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"type": "invalid_data", "message": err.Error()})
		return
	}
	authIdentityID := c.GetString("auth_identity_id")
	user, err := h.Workflow.AcceptInvite(c, token, authIdentityID, input.Email, input.FirstName, input.LastName)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}
