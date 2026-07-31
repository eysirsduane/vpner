package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

const (
	PayConfigOnlyAppleRegions          = "only_apple_regions"
	PayConfigOnlyAppleVersions         = "only_apple_versions"
	PayConfigH5AppleAmount             = "h5_apple_amount"
	PayConfigH5Target                  = "h5_target"
	PayConfigPaidThirdDirectH5         = "paid_third_direct_h5"
	PayConfigOverseasAppleOnly         = "overseas_apple_only"
	PayConfigOverseasTimeZoneAppleOnly = "overseas_timezone_apple_only"
	PayConfigAppleVerifyMode           = "apple.verify_mode"
	PayConfigAppleBundleId             = "apple.bundle_id"
	PayConfigAppleIssuerId             = "apple.issuer_id"
	PayConfigAppleKeyId                = "apple.key_id"
	PayConfigApplePrivateKey           = "apple.private_key"
	PayConfigAppleMockProduct          = "apple.mock_product_id"
	PayConfigXXPayAPIURL               = "xxpay.api_url"
	PayConfigXXPayMchId                = "xxpay.mch_id"
	PayConfigXXPayAppId                = "xxpay.app_id"
	PayConfigXXPayKey                  = "xxpay.key"
	PayConfigXXPayNotifyURL            = "xxpay.notify_url"
	PayConfigXXPayReturnURL            = "xxpay.return_url"
	PayConfigXXPayAlipayID             = "xxpay.alipay_product_id"
	PayConfigXXPayCallbackIPs          = "xxpay.callback_ips"

	legacyConfigPayOnlyAppleRegions  = "pay.only_apple_regions"
	legacyConfigPayOnlyAppleVersions = "pay.only_apple_versions"
	legacyConfigPayH5AppleAmount     = "pay.h5_apple_amount"
)

// PayConfig 支付配置表
type PayConfig struct {
	BaseModel
	Code   string `json:"code" gorm:"column:code;type:varchar(128);uniqueIndex:uniq_code;comment:配置编码"`
	Value  string `json:"value" gorm:"column:value;type:text;comment:配置值"`
	Remark string `json:"remark" gorm:"column:remark;type:varchar(255);comment:说明"`
}

func (PayConfig) TableName() string { return "pay_config" }

func GetPayConfigValue(code string) (string, error) {
	if value, ok := payConfigCache.get(code); ok {
		return value, nil
	}
	var config PayConfig
	err := DB.Where("code = ?", code).First(&config).Error
	if err != nil {
		return "", err
	}
	return config.Value, nil
}

func PayConfigValue(code string, defaultValue string) string {
	value, err := GetPayConfigValue(code)
	if err != nil || value == "" {
		return defaultValue
	}
	return value
}

func InitPayConfigs() error {
	now := time.Now().In(time.Local).Truncate(time.Second)
	configs := []PayConfig{
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      PayConfigOnlyAppleRegions,
			Value:     "",
			Remark:    "仅允许苹果内购的地区列表，多个用英文逗号分隔",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      PayConfigOnlyAppleVersions,
			Value:     "",
			Remark:    "仅允许苹果内购的版本列表，多个用英文逗号分隔",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      PayConfigH5AppleAmount,
			Value:     "0",
			Remark:    "今日苹果内购收款达到多少分后开放H5支付，0表示不限制",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      PayConfigH5Target,
			Value:     "https://www.baidu.com",
			Remark:    "H5支付地址，三方支付接入前默认返回该地址",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      PayConfigPaidThirdDirectH5,
			Value:     "0",
			Remark:    "已成功支付过三方支付的用户是否直接继续走三方支付(0=关闭,1=开启)",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      PayConfigOverseasAppleOnly,
			Value:     "1",
			Remark:    "海外IP是否仅允许苹果内购(0=关闭,1=开启)",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      PayConfigOverseasTimeZoneAppleOnly,
			Value:     "1",
			Remark:    "海外时区是否仅允许苹果内购(0=关闭,1=开启)",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      PayConfigAppleVerifyMode,
			Value:     "production",
			Remark:    "苹果验证模式(mock=测试通过,production=生产验证,sandbox=沙盒验证)",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      PayConfigAppleBundleId,
			Value:     "",
			Remark:    "苹果应用Bundle ID，用于验证交易归属",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      PayConfigAppleIssuerId,
			Value:     "",
			Remark:    "App Store Connect API Issuer ID",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      PayConfigAppleKeyId,
			Value:     "",
			Remark:    "App Store Connect API Key ID",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      PayConfigApplePrivateKey,
			Value:     "",
			Remark:    "App Store Connect API 私钥内容",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      PayConfigAppleMockProduct,
			Value:     "",
			Remark:    "mock模式模拟苹果返回的产品ID",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      PayConfigXXPayAPIURL,
			Value:     "https://dths.hmm2024.xyz",
			Remark:    "XX支付接口域名，不带接口路径",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      PayConfigXXPayMchId,
			Value:     "",
			Remark:    "XX支付商户号",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      PayConfigXXPayAppId,
			Value:     "",
			Remark:    "XX支付应用ID，可为空",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      PayConfigXXPayKey,
			Value:     "",
			Remark:    "XX支付签名私钥",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      PayConfigXXPayNotifyURL,
			Value:     "",
			Remark:    "XX支付异步回调地址，必须是公网可访问地址",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      PayConfigXXPayReturnURL,
			Value:     "",
			Remark:    "XX支付同步跳转地址，可为空",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      PayConfigXXPayAlipayID,
			Value:     "8007",
			Remark:    "XX支付支付宝产品ID，默认支付宝H5支付",
		},
		{
			BaseModel: BaseModel{CreateTime: now},
			Code:      PayConfigXXPayCallbackIPs,
			Value:     "",
			Remark:    "XX支付回调IP白名单，多个用英文逗号分隔，空表示不限制",
		},
	}

	for _, config := range configs {
		config.Value = legacyPayConfigValue(config.Code, config.Value)
		var existing PayConfig
		err := DB.Where("code = ?", config.Code).First(&existing).Error
		if err == nil {
			if existing.Remark == "" && config.Remark != "" {
				if err := DB.Model(&PayConfig{}).Where("id = ?", existing.Id).Update("remark", config.Remark).Error; err != nil {
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

func legacyPayConfigValue(code string, defaultValue string) string {
	legacyCode := ""
	switch code {
	case PayConfigOnlyAppleRegions:
		legacyCode = legacyConfigPayOnlyAppleRegions
	case PayConfigOnlyAppleVersions:
		legacyCode = legacyConfigPayOnlyAppleVersions
	case PayConfigH5AppleAmount:
		legacyCode = legacyConfigPayH5AppleAmount
	default:
		return defaultValue
	}
	value, err := GetConfigValue(legacyCode)
	if err != nil || value == "" {
		return defaultValue
	}
	return value
}
