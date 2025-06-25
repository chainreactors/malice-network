package saas

import (
	"github.com/gin-gonic/gin"
)

// SetupRouter 设置所有saas相关的路由
func SetupRouter(r *gin.Engine) {
	// License 路由
	r.POST("/api/license", CreateLicenseHandler)
	r.GET("/api/license", ListLicensesHandler)
	r.GET("/api/license/:id", GetLicenseHandler)
	r.PUT("/api/license/:id", UpdateLicenseHandler)
	r.DELETE("/api/license/:id", DeleteLicenseHandler)

	// History 路由
	r.POST("/api/history", CreateHistoryHandler)
	r.GET("/api/history", ListHistoriesHandler)
	r.GET("/api/history/:id", GetHistoryHandler)

	// Build 路由
	r.POST("/api/build", BuildHandler)
	r.GET("/api/build/download/:buildname", DownloadBuildHandler)
	r.GET("/api/build/status/:buildname", BuildStatusHandler)
}
