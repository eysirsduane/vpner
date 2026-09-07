package model

const (
	AppleNotificationStatusPending = 1
	AppleNotificationStatusDone    = 2
	AppleNotificationStatusIgnored = 3
	AppleNotificationStatusFailed  = 4
)

// AppleNotification 苹果支付服务端通知表
type AppleNotification struct {
	BaseModel
	NotificationUUID      string `json:"notification_uuid" gorm:"column:notification_uuid;type:varchar(128);uniqueIndex:uniq_notification_uuid;comment:苹果通知唯一ID"`
	NotificationType      string `json:"notification_type" gorm:"column:notification_type;type:varchar(64);index:idx_notification_type;comment:通知类型"`
	Subtype               string `json:"subtype" gorm:"column:subtype;type:varchar(64);comment:通知子类型"`
	Environment           string `json:"environment" gorm:"column:environment;type:varchar(32);comment:环境"`
	TransactionId         string `json:"transaction_id" gorm:"column:transaction_id;type:varchar(128);index:idx_transaction_id;comment:苹果交易号"`
	OriginalTransactionId string `json:"original_transaction_id" gorm:"column:original_transaction_id;type:varchar(128);index:idx_original_transaction_id;comment:苹果原始交易号"`
	ProductId             string `json:"product_id" gorm:"column:product_id;type:varchar(128);comment:苹果产品ID"`
	AppAccountToken       string `json:"app_account_token" gorm:"column:app_account_token;type:varchar(64);index:idx_app_account_token;comment:苹果订单标识"`
	ProcessStatus         int    `json:"process_status" gorm:"column:process_status;type:int;index:idx_process_status;comment:处理状态(1=待处理,2=已处理,3=已忽略,4=处理失败)"`
	ProcessMsg            string `json:"process_msg" gorm:"column:process_msg;type:varchar(255);comment:处理说明"`
	RawPayload            string `json:"raw_payload" gorm:"column:raw_payload;type:longtext;comment:原始通知内容"`
}

func (AppleNotification) TableName() string { return "apple_notification" }
