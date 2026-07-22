package model

const (
	ThirdPayCallbackStatusPending = 1
	ThirdPayCallbackStatusDone    = 2
	ThirdPayCallbackStatusIgnored = 3
	ThirdPayCallbackStatusFailed  = 4
)

// ThirdPayCallback 三方支付回调记录表
type ThirdPayCallback struct {
	BaseModel
	Provider       string `json:"provider" gorm:"column:provider;type:varchar(32);index:idx_provider;comment:支付服务商(xxpay=XX支付)"`
	PayOrderId     string `json:"pay_order_id" gorm:"column:pay_order_id;type:varchar(128);index:idx_pay_order_id;comment:三方支付订单号"`
	MchOrderNo     string `json:"mch_order_no" gorm:"column:mch_order_no;type:varchar(128);index:idx_mch_order_no;comment:商户订单号"`
	MchId          string `json:"mch_id" gorm:"column:mch_id;type:varchar(64);comment:商户号"`
	ProductId      string `json:"product_id" gorm:"column:product_id;type:varchar(64);comment:支付产品ID"`
	Amount         int    `json:"amount" gorm:"column:amount;type:int;comment:订单金额，单位分"`
	Income         int    `json:"income" gorm:"column:income;type:int;comment:实收金额，单位分"`
	Status         string `json:"status" gorm:"column:status;type:varchar(16);index:idx_status;comment:支付状态(-2=订单已关闭,0=订单生成,1=支付中,2=支付成功,3=业务处理完成,4=已退款)"`
	ChannelOrderNo string `json:"channel_order_no" gorm:"column:channel_order_no;type:varchar(128);comment:渠道订单号"`
	PaySuccTime    string `json:"pay_succ_time" gorm:"column:pay_succ_time;type:varchar(32);comment:支付成功时间戳"`
	BackType       string `json:"back_type" gorm:"column:back_type;type:varchar(16);comment:通知类型(1=前台通知,2=后台通知)"`
	ReqTime        string `json:"req_time" gorm:"column:req_time;type:varchar(32);comment:通知请求时间"`
	ClientIp       string `json:"client_ip" gorm:"column:client_ip;type:varchar(64);comment:回调来源IP"`
	Sign           string `json:"sign" gorm:"column:sign;type:varchar(128);comment:签名"`
	SignValid      int    `json:"sign_valid" gorm:"column:sign_valid;type:int;comment:签名是否通过(0=未通过,1=通过)"`
	ProcessStatus  int    `json:"process_status" gorm:"column:process_status;type:int;index:idx_process_status;comment:处理状态(1=待处理,2=已处理,3=已忽略,4=处理失败)"`
	ProcessMsg     string `json:"process_msg" gorm:"column:process_msg;type:varchar(255);comment:处理说明"`
	RawParams      string `json:"raw_params" gorm:"column:raw_params;type:longtext;comment:原始回调参数"`
}

func (ThirdPayCallback) TableName() string { return "third_pay_callback" }
