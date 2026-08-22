package errmsg

import (
	"errors"
	"strings"

	"github.com/go-redis/redis"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

const defaultErrorMessage = "系统错误，请稍后再试"

var exactMessages = map[string]string{
	"invalid json body":                                      "请求参数格式不正确",
	"record not found":                                       "数据不存在",
	"login user not found":                                   "登录状态已失效，请重新登录",
	"authorization token is required":                        "请先登录后再操作",
	"invalid token":                                          "登录状态已失效，请重新登录",
	"token device invalid":                                   "登录设备不匹配，请重新登录",
	"current user missing":                                   "请先登录后再操作",
	"current user invalid":                                   "登录状态异常，请重新登录",
	"user is banned":                                         "当前账号已被限制使用",
	"ban check failed":                                       "登录状态校验失败，请稍后再试",
	"device_no is required":                                  "设备信息不能为空",
	"username is required":                                   "账号不能为空",
	"username format error":                                  "账号格式不正确",
	"password is required":                                   "密码不能为空",
	"password format error":                                  "密码格式不正确",
	"old_password is required":                               "旧密码不能为空",
	"old_password format error":                              "旧密码格式不正确",
	"new_password is required":                               "新密码不能为空",
	"new_password format error":                              "新密码格式不正确",
	"new_password must be different":                         "新密码不能和旧密码相同",
	"account already registered":                             "该账号已注册",
	"account not found":                                      "账号不存在",
	"password error":                                         "账号或密码错误",
	"old_password error":                                     "旧密码不正确",
	"account not logged in":                                  "当前设备未登录账号",
	"current device already logged in":                       "当前设备已登录其他账号",
	"account device limit reached":                           "登录设备数量已达上限，请先移除其它设备",
	"id is required":                                         "请选择要操作的设备",
	"can not remove local device":                            "不能移除当前设备",
	"device not found":                                       "设备不存在",
	"invalid operation":                                      "当前操作不允许",
	"invite_code is required":                                "邀请码不能为空",
	"invite code already used":                               "邀请码已使用",
	"invite code expired":                                    "邀请码填写时间已过期",
	"invite code invalid":                                    "邀请码不存在",
	"can not use local invite code":                          "不能填写自己的邀请码",
	"inviter not found":                                      "邀请人不存在",
	"invite code not available":                              "邀请码暂不可用",
	"invite reward invalid":                                  "邀请奖励暂不可用",
	"invite reward config empty":                             "邀请奖励配置异常，请稍后再试",
	"generate invite code failed":                            "邀请码生成失败，请稍后再试",
	"code is required":                                       "请选择线路地区",
	"vip time expired":                                       "未开通会员或会员已过期，请开通后使用",
	"line is preparing":                                      "线路正在准备中，请稍后再试",
	"please get node first":                                  "请先获取连接节点",
	"connection not found":                                   "当前没有连接记录",
	"flow is required":                                       "流量数据不能为空",
	"flow must be greater than 0":                            "流量数据不正确",
	"node link aes key length must be 16, 24 or 32":          "节点配置异常，请稍后再试",
	"notice id must be greater than 0":                       "请选择要标记的消息",
	"package_id is required":                                 "请选择购买套餐",
	"package not found":                                      "套餐不存在或已下架",
	"package apple_id is required":                           "套餐配置异常，请稍后再试",
	"transaction_id is required":                             "交易信息不能为空",
	"apple product id is required":                           "苹果交易信息异常，请联系客服处理",
	"app_account_token is required":                          "订单信息缺失，请重新发起支付",
	"order not found by app_account_token":                   "订单不存在，请重新发起支付",
	"order not found by empty app_account_token":             "订单信息缺失，请重新发起支付",
	"app_account_token user mismatch":                        "订单账号不匹配，请联系客服处理",
	"apple server api config is required":                    "苹果支付配置异常，请稍后再试",
	"apple transaction id mismatch":                          "苹果交易信息不匹配，请联系客服处理",
	"apple bundle id mismatch":                               "苹果交易信息不匹配，请联系客服处理",
	"apple transaction bundle id mismatch":                   "苹果交易信息不匹配，请联系客服处理",
	"apple transaction revoked":                              "该笔苹果交易已撤销",
	"apple transaction lock timeout":                         "支付处理中，请稍后重试",
	"transaction_id already used":                            "该笔交易已处理，请勿重复提交",
	"transaction_id already refunded":                        "该笔交易已退款",
	"transaction revoked":                                    "该笔交易已撤销",
	"apple product mismatch":                                 "购买商品不匹配，请联系客服处理",
	"order pay type mismatch":                                "订单支付方式不匹配",
	"order already refunded":                                 "订单已退款",
	"xxpay alipay_product_id is required":                    "支付配置异常，请稍后再试",
	"xxpay api url is required":                              "支付配置异常，请稍后再试",
	"xxpay mch_id is required":                               "支付配置异常，请稍后再试",
	"xxpay product_id is required":                           "支付配置异常，请稍后再试",
	"xxpay order_no is required":                             "订单信息异常，请重新发起支付",
	"xxpay amount must be greater than 0":                    "订单金额异常，请重新发起支付",
	"xxpay notify_url is required":                           "支付回调配置异常，请稍后再试",
	"xxpay key is required":                                  "支付配置异常，请稍后再试",
	"xxpay pay url is empty":                                 "支付链接生成失败，请稍后再试",
	"xxpay response sign error":                              "支付结果校验失败，请稍后再试",
	"create xxpay order failed":                              "支付订单创建失败，请稍后再试",
	"query xxpay order failed":                               "支付订单查询失败，请稍后再试",
	"mchOrderNo is required":                                 "订单号不能为空",
	"amount invalid":                                         "支付金额异常",
	"redis is not initialized":                               "服务缓存未就绪，请稍后再试",
	"user_id or device_no is required":                       "封禁信息不完整",
	"ip error":                                               "网络环境异常，请稍后再试",
	"msg, mobile_version and mobile_model_name are required": "请补全错误上报信息",
	"msg_type must be app or vpn":                            "错误类型不正确",
}

var containsMessages = []struct {
	needle string
	msg    string
}{
	{"token is expired", "登录状态已过期，请重新登录"},
	{"Token is expired", "登录状态已过期，请重新登录"},
	{"unexpected signing method", "登录状态异常，请重新登录"},
	{"signature is invalid", "登录状态异常，请重新登录"},
	{"duplicate entry", "数据已存在，请勿重复提交"},
	{"Duplicate entry", "数据已存在，请勿重复提交"},
	{"Error 1062", "数据已存在，请勿重复提交"},
	{"Error 1045", "服务配置异常，请稍后再试"},
	{"Error 1146", "服务数据异常，请稍后再试"},
	{"Error 1292", "服务数据异常，请稍后再试"},
	{"connection refused", "系统连接失败，请稍后再试"},
	{"i/o timeout", "网络连接超时，请稍后再试"},
	{"context deadline exceeded", "请求超时，请稍后再试"},
	{"redis: nil", "服务缓存异常，请稍后再试"},
	{"no such host", "网络连接失败，请稍后再试"},
	{"invalid character", "请求参数格式不正确"},
	{"cannot unmarshal", "请求参数格式不正确"},
	{"Key:", "请求参数不完整或格式不正确"},
	{"Field validation", "请求参数不完整或格式不正确"},
	{"apple", "苹果支付暂不可用，请稍后再试"},
	{"xxpay", "支付服务暂不可用，请稍后再试"},
	{"amount mismatch", "支付金额不匹配，请联系客服处理"},
	{"http status", "支付服务暂不可用，请稍后再试"},
	{"order not found", "订单不存在，请重新发起支付"},
}

// Friendly converts internal errors into short Chinese messages that can be shown to users.
func Friendly(msg string) string {
	msg = strings.TrimSpace(msg)
	if msg == "" {
		return defaultErrorMessage
	}
	if translated, ok := exactMessages[msg]; ok {
		return translated
	}
	if translated := requiredMessage(msg); translated != "" {
		return translated
	}
	for _, item := range containsMessages {
		if strings.Contains(msg, item.needle) {
			return item.msg
		}
	}
	return defaultErrorMessage
}

func FriendlyError(err error) string {
	if err == nil {
		return ""
	}
	if err == gorm.ErrRecordNotFound {
		return exactMessages["record not found"]
	}
	if err == redis.Nil {
		return "服务缓存异常，请稍后再试"
	}
	var mysqlErr *mysql.MySQLError
	if ok := errors.As(err, &mysqlErr); ok {
		switch mysqlErr.Number {
		case 1062:
			return "数据已存在，请勿重复提交"
		case 1045:
			return "服务配置异常，请稍后再试"
		case 1146, 1292:
			return "服务数据异常，请稍后再试"
		default:
			return defaultErrorMessage
		}
	}
	return Friendly(err.Error())
}

func requiredMessage(msg string) string {
	const suffix = " is required"
	if !strings.HasSuffix(msg, suffix) {
		return ""
	}
	field := strings.TrimSuffix(msg, suffix)
	if label := fieldLabel(field); label != "" {
		return label + "不能为空"
	}
	return "请求参数不完整"
}

func fieldLabel(field string) string {
	field = strings.Trim(field, "'\" ")
	labels := map[string]string{
		"platform":          "平台",
		"version":           "版本号",
		"build":             "构建版本号",
		"X-Client-Platform": "平台",
		"X-App-Version":     "版本号",
		"X-Build-No":        "构建版本号",
		"X-Platform":        "平台",
		"X-Version":         "版本号",
		"X-Build":           "构建版本号",
		"device_no":         "设备信息",
		"deviceTag":         "设备信息",
		"username":          "账号",
		"accountName":       "账号",
		"password":          "密码",
		"secret":            "密码",
		"old_password":      "旧密码",
		"oldSecret":         "旧密码",
		"new_password":      "新密码",
		"newSecret":         "新密码",
		"package_id":        "套餐",
		"planKey":           "套餐",
		"transaction_id":    "交易信息",
		"tradeNo":           "交易信息",
		"code":              "线路地区",
		"regionTag":         "线路地区",
		"flow":              "流量数据",
		"usageKb":           "流量数据",
		"invite_code":       "邀请码",
		"referralKey":       "邀请码",
		"id":                "设备",
	}
	return labels[field]
}
