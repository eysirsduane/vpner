package router

import (
	"just-vpn/controller"
	"just-vpn/docs"
	"just-vpn/middleware"
	justswagger "just-vpn/pkg/just-swagger"
	"just-vpn/pkg/mapping"
	"just-vpn/pkg/setting"

	"github.com/gin-gonic/gin"
)

const swaggerDocPath = "/api/v1/docs"

func InitRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.CORSMiddleware())
	router.Use(middleware.ClientInfoMiddleware())

	// 公开路由
	public := router.Group("")
	{
		public.GET(mapping.Endpoint("/api/v1/health"), controller.HealthHandler)                     // 健康检查
		public.POST(mapping.Endpoint("/api/v1/auto_login"), controller.AutoLoginHandler)             // 自动登录
		public.POST("/api/v1/internal/report/stats", controller.InternalReportStatsHandler)          // 内部统计数据
		public.POST("/api/v1/internal/member/lookup", controller.InternalMemberLookupHandler)        // 内部会员查询
		public.POST(mapping.Endpoint("/api/v1/pay/apple_callback"), controller.AppleCallbackHandler) // 苹果支付服务端通知
		public.GET(mapping.Endpoint("/api/v1/pay/xx_callback"), controller.XXPayCallbackHandler)     // XX支付回调
		public.POST(mapping.Endpoint("/api/v1/pay/xx_callback"), controller.XXPayCallbackHandler)    // XX支付回调
		public.GET(mapping.Endpoint("/api/v1/public_apple_id"), controller.PublicAppleIdHandler)     // 获取公共 Apple ID
		public.GET(mapping.Endpoint("/api/v1/upload/*filepath"), controller.UploadFileHandler)       // 获取上传静态资源
	}

	// 用户相关路由
	user := router.Group("").Use(middleware.JWTAuth())
	{
		user.POST(mapping.Endpoint("/api/v1/login"), controller.LoginHandler)                     // 账号登录
		user.POST(mapping.Endpoint("/api/v1/register"), controller.RegisterHandler)               // 注册账号
		user.POST(mapping.Endpoint("/api/v1/logout"), controller.LogoutHandler)                   // 退出登录
		user.POST(mapping.Endpoint("/api/v1/logoff"), controller.LogoffHandler)                   // 账号注销
		user.POST(mapping.Endpoint("/api/v1/change_password"), controller.ChangePasswordHandler)  // 修改密码
		user.POST(mapping.Endpoint("/api/v1/device_info"), controller.DeviceInfoHandler)          // 获取设备信息
		user.POST(mapping.Endpoint("/api/v1/device_logout"), controller.DeviceLogoutHandler)      // 移除登录设备
		user.POST(mapping.Endpoint("/api/v1/draw_invite_code"), controller.DrawInviteCodeHandler) // 获取邀请码
		user.POST(mapping.Endpoint("/api/v1/invite_code"), controller.InviteCodeHandler)          // 填写邀请码
		user.POST(mapping.Endpoint("/api/v1/invite_detail"), controller.InviteDetailHandler)      // 获取邀请详情
		user.GET(mapping.Endpoint("/api/v1/user_info"), controller.UserInfoHandler)               // 获取用户信息
	}

	// 系统相关路由
	system := router.Group("").Use(middleware.JWTAuth())
	{
		system.POST(mapping.Endpoint("/api/v1/init"), controller.InitHandler)                 // 初始化配置
		system.POST(mapping.Endpoint("/api/v1/version"), controller.VersionHandler)           // 版本检测
		system.GET(mapping.Endpoint("/api/v1/advert"), controller.AdvertHandler)              // 获取广告位列表
		system.GET(mapping.Endpoint("/api/v1/notice"), controller.NoticeHandler)              // 获取通知列表
		system.GET(mapping.Endpoint("/api/v1/notice/unread"), controller.UnreadNoticeHandler) // 获取未读通知列表
		system.GET(mapping.Endpoint("/api/v1/popup"), controller.PopupHandler)                // 获取统一弹窗
		system.GET(mapping.Endpoint("/api/v1/delayed_popup"), controller.DelayedPopupHandler) // 获取延迟弹窗
		system.POST(mapping.Endpoint("/api/v1/read_notice"), controller.ReadNoticeHandler)    // 批量标记通知已读
	}

	// 上报相关路由
	report := router.Group("").Use(middleware.JWTAuth())
	{
		report.POST(mapping.Endpoint("/api/v1/error"), controller.ErrorReportHandler) // 错误信息上报
	}

	// 套餐相关路由
	packages := router.Group("").Use(middleware.JWTAuth())
	{
		packages.GET(mapping.Endpoint("/api/v1/packages"), controller.PackagesHandler) // 获取套餐列表
	}

	// 支付相关路由
	pay := router.Group("").Use(middleware.JWTAuth())
	{
		pay.POST(mapping.Endpoint("/api/v1/pay/launch"), controller.PayLaunchHandler)         // 发起支付
		pay.POST(mapping.Endpoint("/api/v1/pay/apple_verify"), controller.AppleVerifyHandler) // 苹果订单验证
	}

	// VPN连接相关路由
	vpn := router.Group("").Use(middleware.JWTAuth())
	{
		vpn.POST(mapping.Endpoint("/api/v1/lines_list"), controller.LinesListHandler)  // 获取国家线路列表
		vpn.POST(mapping.Endpoint("/api/v1/node"), controller.NodeHandler)             // 获取连接节点信息
		vpn.POST(mapping.Endpoint("/api/v1/connected"), controller.ConnectedHandler)   // 确认 VPN 已连接
		vpn.POST(mapping.Endpoint("/api/v1/heartbeat"), controller.HeartbeatHandler)   // VPN 心跳上报
		vpn.POST(mapping.Endpoint("/api/v1/disconnect"), controller.DisconnectHandler) // VPN 断开连接
		vpn.POST(mapping.Endpoint("/api/v1/vpn_flow"), controller.VpnFlowHandler)      // VPN 使用流量上报
	}

	// swagger 文档路由，只在 dev 环境开启
	if setting.AppConfig.RunMode == setting.AppModeDev {
		apiBaseURL := ""
		router.GET(swaggerDocPath+"/*any", justswagger.Handler(justswagger.Config{
			Title:             setting.AppConfig.Product + " API",
			Description:       "API 文档",
			Version:           "1.0.0",
			BasePath:          swaggerDocPath,
			EnableDebug:       true,
			DarkMode:          true,
			UITheme:           justswagger.ThemeMinimal,
			DocJSON:           mapping.MapSwagger(docs.SwaggerInfo.ReadDoc()),
			GlobalHeaders:     mapping.SwaggerGlobalHeaders(),
			TokenExtractRules: mapping.SwaggerTokenExtractRules(),
			Environments: []justswagger.Environment{
				{Name: "本地开发", BaseURL: apiBaseURL},
				{Name: "测试环境", BaseURL: apiBaseURL},
				{Name: "生产环境", BaseURL: apiBaseURL},
			},
		}))
	}

	return router
}
