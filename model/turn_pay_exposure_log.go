package model

// TurnPayExposureLog 支付入口曝光日志表
type TurnPayExposureLog struct {
	BaseModel
	UserId  int    `json:"user_id" gorm:"column:user_id;type:bigint;index:idx_user_create;comment:用户ID"`
	Channel string `json:"channel" gorm:"column:channel;type:varchar(8);index:idx_create_channel;comment:渠道(iap=苹果内购,h5=网页支付)"`
}

func (TurnPayExposureLog) TableName() string { return "turn_pay_exposure_log" }
