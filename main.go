package main

import (
	"fmt"

	"just-vpn/model"
	"just-vpn/pkg/redis"
	"just-vpn/pkg/setting"
	"just-vpn/router"
	"just-vpn/task"

	"github.com/gin-gonic/gin"
)

// @title Just VPN API
// @version 1.0
// @description Just VPN 后端接口文档
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	gin.SetMode(setting.GinMode())

	if err := model.Init(); err != nil {
		panic(err)
	}
	if err := redis.InitRedis(setting.RedisConfig.Host, setting.RedisConfig.Password, 0); err != nil {
		panic(err)
	}
	if err := model.LoadActiveUserBansToCache(); err != nil {
		panic(err)
	}
	model.StartNodeOnlineCleaner()
	task.StartLogCleanupTask()
	task.StartDailyStatTask()
	task.StartShowRecordFlushTask()
	task.StartNodePullTask()

	r := router.InitRouter()
	addr := fmt.Sprintf(":%d", setting.AppConfig.HttpPort)
	if err := r.Run(addr); err != nil {
		panic(err)
	}
}
