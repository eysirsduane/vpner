package errmsg

import "strings"

var englishMessages = map[string]string{
	"系统错误，请稍后再试":          "System error. Please try again later.",
	"请求参数格式不正确":           "Invalid request format.",
	"数据不存在":               "Data not found.",
	"登录状态已失效，请重新登录":       "Your login session is no longer valid. Please sign in again.",
	"请先登录后再操作":            "Please sign in first.",
	"登录设备不匹配，请重新登录":       "This device does not match your login session. Please sign in again.",
	"登录状态异常，请重新登录":        "Your login session is invalid. Please sign in again.",
	"当前账号已被限制使用":          "This account has been restricted.",
	"登录状态校验失败，请稍后再试":      "Unable to verify your login status. Please try again later.",
	"设备信息不能为空":            "Device information is required.",
	"账号不能为空":              "Account is required.",
	"账号格式不正确":             "Invalid account format.",
	"密码不能为空":              "Password is required.",
	"密码格式不正确":             "Invalid password format.",
	"旧密码不能为空":             "Current password is required.",
	"旧密码格式不正确":            "Invalid current password format.",
	"新密码不能为空":             "New password is required.",
	"新密码格式不正确":            "Invalid new password format.",
	"新密码不能和旧密码相同":         "The new password must be different from the current password.",
	"该账号已注册":              "This account is already registered.",
	"账号不存在":               "Account not found.",
	"账号或密码错误":             "Incorrect account or password.",
	"旧密码不正确":              "The current password is incorrect.",
	"当前设备未登录账号":           "No account is signed in on this device.",
	"当前设备已登录其他账号":         "Another account is already signed in on this device.",
	"登录设备数量已达上限，请先移除其它设备": "The device limit has been reached. Remove another device first.",
	"请选择要操作的设备":           "Please select a device.",
	"不能移除当前设备":            "The current device cannot be removed.",
	"设备不存在":               "Device not found.",
	"当前操作不允许":             "This operation is not allowed.",
	"邀请码不能为空":             "Invitation code is required.",
	"邀请码已使用":              "The invitation code has already been used.",
	"邀请码填写时间已过期":          "The invitation code entry period has expired.",
	"邀请码不存在":              "Invitation code not found.",
	"不能填写自己的邀请码":          "You cannot use your own invitation code.",
	"邀请人不存在":              "Inviter not found.",
	"邀请码暂不可用":             "The invitation code is currently unavailable.",
	"邀请奖励暂不可用":            "The invitation reward is currently unavailable.",
	"邀请奖励配置异常，请稍后再试":      "Invitation reward configuration error. Please try again later.",
	"邀请码生成失败，请稍后再试":       "Unable to generate an invitation code. Please try again later.",
	"请选择线路地区":             "Please select a server region.",
	"未开通会员或会员已过期，请开通后使用":  "Membership is required or has expired. Please subscribe to continue.",
	"线路正在准备中，请稍后再试":       "The server is being prepared. Please try again later.",
	"请先获取连接节点":            "Please get a server node first.",
	"当前没有连接记录":            "No active connection record was found.",
	"流量数据不能为空":            "Traffic data is required.",
	"流量数据不正确":             "Invalid traffic data.",
	"节点配置异常，请稍后再试":        "Server configuration error. Please try again later.",
	"请选择要标记的消息":           "Please select a notification.",
	"请选择购买套餐":             "Please select a plan.",
	"套餐不存在或已下架":           "The selected plan is unavailable.",
	"套餐配置异常，请稍后再试":        "Plan configuration error. Please try again later.",
	"交易信息不能为空":            "Transaction information is required.",
	"苹果交易信息异常，请联系客服处理":    "Apple transaction information is invalid. Please contact support.",
	"订单信息缺失，请重新发起支付":      "Order information is missing. Please start the payment again.",
	"订单不存在，请重新发起支付":       "Order not found. Please start the payment again.",
	"订单账号不匹配，请联系客服处理":     "The order does not belong to this account. Please contact support.",
	"苹果支付配置异常，请稍后再试":      "Apple payment configuration error. Please try again later.",
	"苹果交易信息不匹配，请联系客服处理":   "Apple transaction information does not match. Please contact support.",
	"该笔苹果交易已撤销":           "This Apple transaction has been revoked.",
	"支付处理中，请稍后重试":         "Payment is being processed. Please try again shortly.",
	"该笔交易已处理，请勿重复提交":      "This transaction has already been processed.",
	"该笔交易已退款":             "This transaction has been refunded.",
	"该笔交易已撤销":             "This transaction has been revoked.",
	"购买商品不匹配，请联系客服处理":     "The purchased product does not match. Please contact support.",
	"订单支付方式不匹配":           "The order payment method does not match.",
	"订单已退款":               "The order has been refunded.",
	"支付配置异常，请稍后再试":        "Payment configuration error. Please try again later.",
	"订单信息异常，请重新发起支付":      "Order information is invalid. Please start the payment again.",
	"订单金额异常，请重新发起支付":      "The order amount is invalid. Please start the payment again.",
	"支付回调配置异常，请稍后再试":      "Payment callback configuration error. Please try again later.",
	"支付链接生成失败，请稍后再试":      "Unable to create the payment link. Please try again later.",
	"支付结果校验失败，请稍后再试":      "Unable to verify the payment result. Please try again later.",
	"支付订单创建失败，请稍后再试":      "Unable to create the payment order. Please try again later.",
	"支付订单查询失败，请稍后再试":      "Unable to query the payment order. Please try again later.",
	"订单号不能为空":             "Order number is required.",
	"支付金额异常":              "Invalid payment amount.",
	"服务缓存未就绪，请稍后再试":       "The service cache is not ready. Please try again later.",
	"封禁信息不完整":             "Restriction information is incomplete.",
	"网络环境异常，请稍后再试":        "Network environment error. Please try again later.",
	"请补全错误上报信息":           "Please provide all required error report information.",
	"错误类型不正确":             "Invalid error type.",
	"登录状态已过期，请重新登录":       "Your login session has expired. Please sign in again.",
	"数据已存在，请勿重复提交":        "This data already exists. Please do not submit it again.",
	"服务配置异常，请稍后再试":        "Service configuration error. Please try again later.",
	"服务数据异常，请稍后再试":        "Service data error. Please try again later.",
	"系统连接失败，请稍后再试":        "Unable to connect to the service. Please try again later.",
	"网络连接超时，请稍后再试":        "Network connection timed out. Please try again later.",
	"请求超时，请稍后再试":          "The request timed out. Please try again later.",
	"服务缓存异常，请稍后再试":        "Service cache error. Please try again later.",
	"网络连接失败，请稍后再试":        "Network connection failed. Please try again later.",
	"请求参数不完整或格式不正确":       "Required request parameters are missing or invalid.",
	"苹果支付暂不可用，请稍后再试":      "Apple payment is temporarily unavailable. Please try again later.",
	"支付服务暂不可用，请稍后再试":      "Payment service is temporarily unavailable. Please try again later.",
	"支付金额不匹配，请联系客服处理":     "The payment amount does not match. Please contact support.",
	"请求参数不完整":             "A required request parameter is missing.",
	"平台不能为空":              "Platform is required.",
	"版本号不能为空":             "App version is required.",
	"构建版本号不能为空":           "Build number is required.",
	"套餐不能为空":              "Plan is required.",
	"线路地区不能为空":            "Server region is required.",
	"设备不能为空":              "Device is required.",
	"会员已过期，请断开连接":         "Membership has expired. Please disconnect.",
	"本机设备":                "This device",
}

// Localize returns a translated hard-coded user-facing message. Unknown
// strings are kept unchanged so protocol values and diagnostic details are not
// accidentally rewritten.
func Localize(msg string, language string) string {
	if isChineseLanguage(language) {
		return msg
	}
	if translated, ok := englishMessages[msg]; ok {
		return translated
	}
	return msg
}

func FriendlyForLanguage(msg string, language string) string {
	return Localize(Friendly(msg), language)
}

func isChineseLanguage(language string) bool {
	language = strings.ToLower(strings.TrimSpace(language))
	return language == "" || language == "zh" || strings.HasPrefix(language, "zh-") || strings.HasPrefix(language, "zh_")
}
