package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

const (
	InviteConfigValidHours    = "invite_valid_hours"
	InviteConfigRewardRules   = "invite_reward_rules"
	InviteConfigMaxRewardDays = "invite_max_reward_days"

	defaultInviteRewardRules   = `{"per_invite_seconds":3600,"milestones":[{"count":24,"reward_seconds":2592000}]}`
	legacyDefaultInviteRewards = `[{"min_count":1,"reward_seconds":3600},{"min_count":5,"reward_seconds":86400},{"min_count":20,"reward_seconds":2592000}]`
)

// InviteConfig 邀请码配置表
type InviteConfig struct {
	BaseModel
	ConfigKey   string `json:"config_key" gorm:"column:config_key;type:varchar(128);uniqueIndex:uniq_config_key;comment:配置Key"`
	ConfigValue string `json:"config_value" gorm:"column:config_value;type:text;comment:配置值"`
	Remark      string `json:"remark" gorm:"column:remark;type:varchar(255);comment:说明"`
}

func (InviteConfig) TableName() string { return "invite_config" }

func GetInviteConfigValue(key string) (string, error) {
	if value, ok := inviteConfigCache.get(key); ok {
		return value, nil
	}
	var config InviteConfig
	err := DB.Where("config_key = ?", key).First(&config).Error
	if err != nil {
		return "", err
	}
	return config.ConfigValue, nil
}

func InitInviteConfigs() error {
	now := time.Now().In(time.Local).Truncate(time.Second)
	configs := []InviteConfig{
		{
			BaseModel:   BaseModel{CreateTime: now},
			ConfigKey:   InviteConfigValidHours,
			ConfigValue: "72",
			Remark:      "新用户注册后可填写邀请码的小时数",
		},
		{
			BaseModel:   BaseModel{CreateTime: now},
			ConfigKey:   InviteConfigRewardRules,
			ConfigValue: defaultInviteRewardRules,
			Remark:      "邀请奖励规则，每邀请一人固定奖励，满指定人数额外奖励",
		},
		{
			BaseModel:   BaseModel{CreateTime: now},
			ConfigKey:   InviteConfigMaxRewardDays,
			ConfigValue: "300",
			Remark:      "邀请累计最大奖励天数",
		},
	}

	for _, config := range configs {
		var existing InviteConfig
		err := DB.Where("config_key = ?", config.ConfigKey).First(&existing).Error
		if err == nil {
			if existing.ConfigKey == InviteConfigRewardRules && existing.ConfigValue == legacyDefaultInviteRewards {
				if err := DB.Model(&InviteConfig{}).Where("id = ?", existing.Id).Updates(map[string]interface{}{
					"config_value": config.ConfigValue,
					"remark":       config.Remark,
				}).Error; err != nil {
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
