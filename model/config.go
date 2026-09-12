package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

const (
	ConfigUserAgreement              = "user_agreement"
	ConfigPrivacyAgreement           = "privacy_agreement"
	ConfigShareQrcode                = "share_qrcode"
	ConfigShareLinks                 = "share.links"
	ConfigWebsite                    = "website"
	ConfigNewUserFreeSeconds         = "new_user_free_seconds"
	ConfigToolCustomer               = "tool.customer_service"
	ConfigToolCustomerServiceEnabled = "tool.customer_service_enabled"
	ConfigToolCustomerServiceURL     = "tool.customer_service_url"
	ConfigToolCleanMemoryEnabled     = "tool.clean_memory_enabled"
	ConfigToolMessageEnabled         = "tool.message_enabled"
	ConfigNodeLinkAESKey             = "node.link_aes_key"
	ConfigNodePullURL                = "node.pull_url"
	ConfigNodePullIntervalSeconds    = "node.pull_interval_seconds"
	ConfigNodeReviewVersions         = "node.review_versions"
	ConfigNodeReviewLink             = "node.review_link"
	ConfigSkipProxyDomains           = "skip_proxy_domains"
	ConfigRealLogoffVersions         = "account.real_logoff_versions"
	ConfigMaxLoginDevices            = "account.max_login_devices"
	ConfigRequestResponseDebug       = "debug.request_response_enabled"
	ConfigRequestResponseDebugUsers  = "debug.request_response_user_ids"
)

// Config 系统配置表
type Config struct {
	BaseModel
	Code   string `json:"code" gorm:"column:code;type:varchar(128);uniqueIndex:uniq_code;comment:配置编码"`
	Value  string `json:"value" gorm:"column:value;type:text;comment:配置值"`
	Remark string `json:"remark" gorm:"column:remark;type:varchar(255);comment:说明"`
}

func (Config) TableName() string { return "config" }

func GetConfigValue(code string) (string, error) {
	if value, ok := systemConfigCache.get(code); ok {
		return value, nil
	}
	var config Config
	err := DB.Where("code = ?", code).First(&config).Error
	if err != nil {
		return "", err
	}
	return config.Value, nil
}

func ConfigValue(code string, defaultValue string) string {
	value, err := GetConfigValue(code)
	if err != nil || value == "" {
		return defaultValue
	}
	return value
}

func InitConfigs() error {
	now := time.Now().In(time.Local).Truncate(time.Second)
	oldCustomerServiceURL := ""
	if value, err := GetConfigValue(ConfigToolCustomer); err == nil {
		oldCustomerServiceURL = value
	}
	customerServiceEnabledDefault := "off"
	if oldCustomerServiceURL != "" {
		customerServiceEnabledDefault = "on"
	}
	configs := []Config{
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      ConfigUserAgreement,
			Value:     "",
			Remark:    "用户服务协议地址",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      ConfigPrivacyAgreement,
			Value:     "",
			Remark:    "隐私协议地址",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      ConfigShareQrcode,
			Value:     "",
			Remark:    "分享二维码图片地址",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      ConfigShareLinks,
			Value:     "[]",
			Remark:    "分享链接列表，JSON数组格式",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      ConfigWebsite,
			Value:     "",
			Remark:    "官网地址",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      ConfigNewUserFreeSeconds,
			Value:     "600",
			Remark:    "新用户首次自动登录赠送会员秒数，0表示不赠送",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      ConfigToolCustomerServiceEnabled,
			Value:     customerServiceEnabledDefault,
			Remark:    "在线客服入口开关，on开启，off关闭",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      ConfigToolCustomerServiceURL,
			Value:     oldCustomerServiceURL,
			Remark:    "在线客服地址",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      ConfigToolCleanMemoryEnabled,
			Value:     "off",
			Remark:    "清理内存入口开关，on开启，off关闭",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      ConfigToolMessageEnabled,
			Value:     "on",
			Remark:    "消息通知入口开关，on开启，off关闭",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      ConfigNodeLinkAESKey,
			Value:     "g9rsnoih20vuiqpw",
			Remark:    "节点URL AES加密key，长度必须为16、24或32位",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      ConfigNodePullURL,
			Value:     "",
			Remark:    "节点订阅拉取地址，留空表示不自动同步",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      ConfigNodePullIntervalSeconds,
			Value:     "300",
			Remark:    "节点拉取时间间隔，单位秒，无效或非正数时使用300秒",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      ConfigNodeReviewVersions,
			Value:     "",
			Remark:    "审核版本列表，多个用英文逗号分隔，留空表示关闭审核节点",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      ConfigNodeReviewLink,
			Value:     "",
			Remark:    "审核版本固定使用的节点字符串",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      ConfigSkipProxyDomains,
			Value:     "",
			Remark:    "不走代理的域名列表，多个用逗号分隔",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      ConfigRealLogoffVersions,
			Value:     "",
			Remark:    "真实注销账号版本列表，多个用英文逗号分隔，空值表示所有版本只返回成功",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      ConfigMaxLoginDevices,
			Value:     "2",
			Remark:    "每个账号允许同时登录的最大设备数",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      ConfigRequestResponseDebug,
			Value:     "off",
			Remark:    "指定用户请求响应调试日志开关，on开启，off关闭",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      ConfigRequestResponseDebugUsers,
			Value:     "",
			Remark:    "请求响应调试用户ID列表，多个用英文逗号分隔",
		},
	}

	for _, config := range configs {
		var existing Config
		err := DB.Where("code = ?", config.Code).First(&existing).Error
		if err == nil {
			if config.Code == ConfigNewUserFreeSeconds && existing.Value == "0" && existing.Remark == "新用户默认赠送会员秒数" {
				if err := DB.Model(&Config{}).Where("id = ?", existing.Id).Updates(map[string]interface{}{
					"value":  config.Value,
					"remark": config.Remark,
				}).Error; err != nil {
					return err
				}
				continue
			}
			if config.Code == ConfigToolCustomerServiceURL && existing.Value == "" && oldCustomerServiceURL != "" {
				if err := DB.Model(&Config{}).Where("id = ?", existing.Id).Updates(map[string]interface{}{
					"value":  oldCustomerServiceURL,
					"remark": config.Remark,
				}).Error; err != nil {
					return err
				}
				continue
			}
			if existing.Remark == "" && config.Remark != "" {
				if err := DB.Model(&Config{}).Where("id = ?", existing.Id).Update("remark", config.Remark).Error; err != nil {
					return err
				}
			}
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := DB.Create(&config).Error; err != nil {
			return err
		}
	}
	return nil
}
