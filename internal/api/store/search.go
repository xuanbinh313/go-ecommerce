package store

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Store Search requires Medusa's configureStoreSearch allowlist and provider
// index field validation. Until that config is wired, the route preserves the
// source disabled-module failure rather than serving unscoped index contents.
type SearchHTTP struct{}

func (h *SearchHTTP) Register(router *gin.RouterGroup, authenticate gin.HandlerFunc) {
	router.POST("/store/search", authenticate, h.searchUnavailable)
}

func (h *SearchHTTP) searchUnavailable(c *gin.Context) {
	c.JSON(http.StatusBadRequest, gin.H{"type": "not_allowed", "message": "The Search Module is not installed, so nothing can be searched."})
}
