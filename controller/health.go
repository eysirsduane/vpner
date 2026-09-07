package controller

import "github.com/gin-gonic/gin"

// HealthHandler 健康检查
// @Summary 健康检查
// @Description 检查 API 服务是否正常运行
// @Tags 系统
// @Produce json
// @Success 200 {object} Response
// @Router /health [get]
func HealthHandler(c *gin.Context) {
	JsonReturn(c, CodeSuccess, "success", nil)
}
