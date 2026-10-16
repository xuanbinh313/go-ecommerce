package store

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/example/medusa-go-clone/internal/modules/order/domain/models"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type ReturnReasonReader interface {
	ListReturnReasons(context.Context, models.ReturnReasonFilter, int, int) ([]models.ReturnReason, int, error)
	GetReturnReason(context.Context, string) (models.ReturnReason, error)
}

type ReturnReasonHTTP struct{ Service ReturnReasonReader }

func (h *ReturnReasonHTTP) Register(router *gin.RouterGroup) {
	router.GET("/store/return-reasons", h.list)
	router.GET("/store/return-reasons/:id", h.get)
}
func (h *ReturnReasonHTTP) list(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit < 1 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	rows, count, err := h.Service.ListReturnReasons(c, models.ReturnReasonFilter{}, limit, offset)
	if err != nil {
		writeReturnReasonError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"return_reasons": rows, "count": count, "offset": offset, "limit": limit})
}
func (h *ReturnReasonHTTP) get(c *gin.Context) {
	row, err := h.Service.GetReturnReason(c, c.Param("id"))
	if err != nil {
		writeReturnReasonError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"return_reason": row})
}
func writeReturnReasonError(c *gin.Context, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"type": "not_found", "message": "Return reason not found"})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"type": "database_error", "message": err.Error()})
}
