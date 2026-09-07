package controller

import (
	"encoding/json"
	"strconv"
	"strings"

	"just-vpn/middleware"
	"just-vpn/model"

	"github.com/gin-gonic/gin"
)

// InitAgreementsResponse 协议配置
type InitAgreementsResponse struct {
	UserAgreement    string `json:"user_agreement" example:"https://example.com/user-agreement"`    // 用户服务协议地址
	PrivacyAgreement string `json:"privacy_agreement" example:"https://example.com/privacy-policy"` // 隐私协议地址
}

// InitShareResponse 分享配置
type InitShareResponse struct {
	Qrcode string   `json:"qrcode" example:"https://example.com/share.png"` // 分享二维码图片地址
	Links  []string `json:"links" example:"https://example.com/a"`          // 分享链接列表
}

// InitInviteMilestoneResponse 邀请阶梯奖励配置
type InitInviteMilestoneResponse struct {
	Count         int    `json:"count" example:"24"`               // 达到邀请人数
	RewardSeconds int    `json:"reward_seconds" example:"2592000"` // 额外奖励会员秒数
	RewardText    string `json:"reward_text" example:"1个月会员"`      // 额外奖励会员时长文案
}

// InitInviteResponse 邀请配置
type InitInviteResponse struct {
	ValidHours       int                           `json:"valid_hours" example:"72"`          // 新用户注册后可填写邀请码的小时数
	PerInviteSeconds int                           `json:"per_invite_seconds" example:"3600"` // 每邀请一个用户奖励的会员秒数
	PerInviteText    string                        `json:"per_invite_text" example:"1小时会员"`   // 每邀请一个用户奖励的会员时长文案
	MaxRewardDays    int                           `json:"max_reward_days" example:"300"`     // 邀请累计最大奖励天数
	Milestones       []InitInviteMilestoneResponse `json:"milestones"`                        // 邀请人数阶梯奖励列表
}

// InitAppResponse 应用配置
type InitAppResponse struct {
	Website            string `json:"website" example:"https://example.com"` // 官网地址
	NewUserFreeSeconds int    `json:"new_user_free_seconds" example:"3600"`  // 新用户默认赠送会员秒数
}

// InitToolsResponse 工具入口配置
type InitToolsResponse struct {
	CustomerServiceEnabled string `json:"customer_service_enabled" example:"on"`                   // 在线客服入口开关(on=开启,off=关闭)
	CustomerServiceUrl     string `json:"customer_service_url" example:"https://example.com/chat"` // 在线客服地址，配置模板中的#ID会替换为当前用户ID
	CleanMemoryEnabled     string `json:"clean_memory_enabled" example:"on"`                       // 清理内存入口开关(on=开启,off=关闭)
	MessageEnabled         string `json:"message_enabled" example:"on"`                            // 消息通知入口开关(on=开启,off=关闭)
}

// InitProxyResponse 代理配置
type InitProxyResponse struct {
	SkipDomains []string `json:"skip_domains" example:"example.com"` // 不走代理的域名列表
}

// InitResponse 初始化配置响应
type InitResponse struct {
	Agreements InitAgreementsResponse `json:"agreements"` // 协议配置
	Share      InitShareResponse      `json:"share"`      // 分享配置
	Invite     InitInviteResponse     `json:"invite"`     // 邀请配置
	App        InitAppResponse        `json:"app"`        // 应用配置
	Tools      InitToolsResponse      `json:"tools"`      // 工具入口配置
	Proxy      InitProxyResponse      `json:"proxy"`      // 代理配置
}

// InitAPIResponse 初始化配置接口响应
type InitAPIResponse struct {
	Code   int          `json:"code" example:"200"`    // 响应状态码
	Msg    string       `json:"msg" example:"success"` // 响应消息
	Result InitResponse `json:"result"`                // 初始化配置
}

func configInt(code string, defaultValue int) int {
	value := model.ConfigValue(code, "")
	if value == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return n
}

func configSwitch(code string, defaultValue string) string {
	value := strings.ToLower(strings.TrimSpace(model.ConfigValue(code, defaultValue)))
	if value == "on" {
		return "on"
	}
	return "off"
}

func inviteConfigValueInt(code string, defaultValue int) int {
	value, err := model.GetInviteConfigValue(code)
	if err != nil || value == "" {
		return defaultValue
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return n
}

func splitConfigList(value string) []string {
	if value == "" {
		return []string{}
	}
	items := strings.Split(value, ",")
	result := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func configStringJSONList(code string) []string {
	value := strings.TrimSpace(model.ConfigValue(code, "[]"))
	if value == "" {
		return []string{}
	}
	var list []string
	if err := json.Unmarshal([]byte(value), &list); err != nil {
		return []string{}
	}
	result := make([]string, 0, len(list))
	for _, item := range list {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func buildInitInvite(language string) InitInviteResponse {
	validHours := inviteConfigValueInt(model.InviteConfigValidHours, 72)
	maxRewardDays := inviteConfigValueInt(model.InviteConfigMaxRewardDays, 300)
	config, milestones, err := inviteRewardDetail(language)
	if err != nil {
		config = inviteRewardConfig{
			PerInviteSeconds: 3600,
		}
		milestones = []InviteMilestoneResponse{
			{
				Count:         24,
				RewardSeconds: 2592000,
				RewardText:    rewardText(2592000, language),
			},
		}
	}

	initMilestones := make([]InitInviteMilestoneResponse, 0, len(milestones))
	for _, milestone := range milestones {
		initMilestones = append(initMilestones, InitInviteMilestoneResponse{
			Count:         milestone.Count,
			RewardSeconds: milestone.RewardSeconds,
			RewardText:    milestone.RewardText,
		})
	}

	return InitInviteResponse{
		ValidHours:       validHours,
		PerInviteSeconds: config.PerInviteSeconds,
		PerInviteText:    rewardText(config.PerInviteSeconds, language),
		MaxRewardDays:    maxRewardDays,
		Milestones:       initMilestones,
	}
}

func buildCustomerServiceURL(user model.User) string {
	link := strings.TrimSpace(model.ConfigValue(model.ConfigToolCustomerServiceURL, ""))
	if link == "" {
		return ""
	}
	return strings.ReplaceAll(link, "#ID", strconv.Itoa(user.Id))
}

func buildInitResponse(user model.User, language string) InitResponse {
	return InitResponse{
		Agreements: InitAgreementsResponse{
			UserAgreement:    model.ConfigValue(model.ConfigUserAgreement, ""),
			PrivacyAgreement: model.ConfigValue(model.ConfigPrivacyAgreement, ""),
		},
		Share: InitShareResponse{
			Qrcode: model.ConfigValue(model.ConfigShareQrcode, ""),
			Links:  configStringJSONList(model.ConfigShareLinks),
		},
		Invite: buildInitInvite(language),
		App: InitAppResponse{
			Website:            model.ConfigValue(model.ConfigWebsite, ""),
			NewUserFreeSeconds: configInt(model.ConfigNewUserFreeSeconds, 3600),
		},
		Tools: InitToolsResponse{
			CustomerServiceEnabled: configSwitch(model.ConfigToolCustomerServiceEnabled, "off"),
			CustomerServiceUrl:     buildCustomerServiceURL(user),
			CleanMemoryEnabled:     configSwitch(model.ConfigToolCleanMemoryEnabled, "off"),
			MessageEnabled:         configSwitch(model.ConfigToolMessageEnabled, "on"),
		},
		Proxy: InitProxyResponse{
			SkipDomains: splitConfigList(model.ConfigValue(model.ConfigSkipProxyDomains, "")),
		},
	}
}

// InitHandler 初始化配置
// @Summary 初始化配置
// @Description 获取客户端启动所需的协议、分享、邀请、应用、工具入口和代理配置
// @Tags 系统
// @Produce json
// @Security BearerAuth
// @Success 200 {object} InitAPIResponse
// @Router /init [post]
func InitHandler(c *gin.Context) {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	language := middleware.CurrentClientInfo(c).Language
	JsonReturn(c, CodeSuccess, "success", buildInitResponse(user, language))
}
