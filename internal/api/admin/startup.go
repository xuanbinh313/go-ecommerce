package admin

import "github.com/gin-gonic/gin"

// RegisterStartupRoutes exposes the public metadata endpoints requested while
// the Medusa dashboard initializes. Other Admin API routes are not ported yet.
func RegisterStartupRoutes(router *gin.RouterGroup, featureFlags ...map[string]bool) {
	router.GET("/admin/feature-flags", func(c *gin.Context) {
		flags:=gin.H{}
		if len(featureFlags)>0{for key,value:=range featureFlags[0]{flags[key]=value}}
		c.JSON(200, gin.H{"feature_flags": flags})
	})
	router.GET("/cloud/auth", func(c *gin.Context) {
		c.JSON(200, gin.H{"enabled": false})
	})
}
