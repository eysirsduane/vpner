package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

const (
	OrderPayStatusUnpaid = 1
	OrderPayStatusPaid   = 3
	OrderPayStatusRefund = 4

	OrderPayTypeAppleIAP = "apple_iap"
	OrderPayTypeH5       = "h5"
	OrderPayTypeXXPay    = "xxpay"
	OrderPayTypeSSPay    = "sspay"

	OrderPaySourceLaunch       = "launch"
	OrderPaySourceAppleVerify  = "apple_verify"
	OrderPaySourceAppleS2S     = "apple_s2s"
	OrderPaySourceThirdPayPage = "third_pay_page"
	OrderPaySourceXXCallback   = "xx_callback"
	OrderPaySourceXXQuery      = "xx_query"
	OrderPaySourceSSCallback   = "ss_callback"
	OrderPaySourceSSQuery      = "ss_query"
)

// Order 订单表
type Order struct {
	BaseModel
	PakId           int        `json:"pak_id" gorm:"column:pak_id;type:int;comment:套餐ID"`
	PakName         string     `json:"pak_name" gorm:"column:pak_name;type:varchar(128);comment:套餐名称"`
	PakTime         int        `json:"pak_time" gorm:"column:pak_time;type:int;comment:购买套餐增加秒数"`
	OrderNo         string     `json:"order_no" gorm:"column:order_no;type:varchar(128);index:idx_order_no;comment:订单号"`
	Name            string     `json:"name" gorm:"column:name;type:varchar(128);comment:名称"`
	PayStatus       int        `json:"pay_status" gorm:"column:pay_status;type:int;index:idx_pay_status;comment:支付状态(1=未支付,3=已支付,4=已退款)"`
	PayType         string     `json:"pay_type" gorm:"column:pay_type;type:varchar(32);comment:支付方式"`
	PayReason       string     `json:"pay_reason" gorm:"column:pay_reason;type:varchar(128);comment:支付分流原因"`
	ClientTimeZone  string     `json:"client_time_zone" gorm:"column:client_time_zone;type:varchar(128);comment:发起支付时客户端时区"`
	PayProductId    string     `json:"pay_product_id" gorm:"column:pay_product_id;type:varchar(128);comment:支付产品ID"`
	AppAccountToken string     `json:"app_account_token" gorm:"column:app_account_token;type:varchar(64);index:idx_app_account_token;comment:苹果内购订单标识"`
	Uid             int        `json:"uid" gorm:"column:uid;type:int;index:idx_uid;comment:用户ID"`
	RegPlatform     string     `json:"reg_platform" gorm:"column:reg_platform;type:varchar(32);comment:用户注册平台"`
	RegTime         time.Time  `json:"reg_time" gorm:"column:reg_time;type:datetime;comment:用户注册时间"`
	Price           int        `json:"price" gorm:"column:price;type:int;comment:订单金额，单位分"`
	Money           int        `json:"money" gorm:"column:money;type:int;comment:实际支付金额，单位分"`
	PaySn           string     `json:"pay_sn" gorm:"column:pay_sn;type:varchar(128);index:idx_pay_sn;comment:支付平台订单号"`
	PayUrl          string     `json:"pay_url" gorm:"column:pay_url;type:text;comment:H5支付链接"`
	Origin          string     `json:"origin" gorm:"column:origin;type:varchar(32);comment:来源"`
	Ip              string     `json:"ip" gorm:"column:ip;type:varchar(64);comment:IP地址"`
	OrderFlag       int        `json:"order_flag" gorm:"column:order_flag;type:int;comment:订单标记(1=新购,2=续费,3=复活)"`
	PaySource       string     `json:"pay_source" gorm:"column:pay_source;type:varchar(64);comment:支付来源(client_verify=客户端验证,s2s_callback=服务端回调,restore=恢复购买,admin_manual=后台手动,good_callback=Good支付回调,yb_callback=易宝回调,xx_callback=星星回调,xx_query=星星查询)"`
	VerifyTime      *time.Time `json:"verify_time" gorm:"column:verify_time;type:datetime;comment:凭据提交时间"`
	PayTime         *time.Time `json:"pay_time" gorm:"column:pay_time;type:datetime;comment:支付时间"`
	SourceChannel   string     `json:"source_channel" gorm:"column:source_channel;type:varchar(128);comment:来源渠道"`
	RegIpRegion     string     `json:"reg_ip_region" gorm:"column:reg_ip_region;type:varchar(255);comment:下单IP区域"`
	Sign            string     `json:"sign" gorm:"column:sign;type:varchar(255);comment:数据签名"`
}

func (Order) TableName() string { return "order" }

func CreateOrder(order *Order) error {
	return DB.Create(order).Error
}

func GetOrderByOrderNo(orderNo string) (Order, error) {
	var order Order
	err := DB.Where("order_no = ?", orderNo).First(&order).Error
	return order, err
}

func GetAppleOrderByTransactionId(transactionId string) (Order, error) {
	var order Order
	if transactionId == "" {
		return order, gorm.ErrRecordNotFound
	}
	err := DB.Where("pay_type = ?", OrderPayTypeAppleIAP).
		Where("pay_status IN ?", []int{OrderPayStatusPaid, OrderPayStatusRefund}).
		Where("pay_sn = ?", transactionId).
		First(&order).Error
	return order, err
}

func GetAppleOrderByAppAccountToken(token string) (Order, error) {
	var order Order
	if token == "" {
		return order, gorm.ErrRecordNotFound
	}
	err := DB.Where("pay_type = ?", OrderPayTypeAppleIAP).
		Where("app_account_token = ?", token).
		First(&order).Error
	return order, err
}

func IsOrderNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

func TodayApplePaidAmount(now time.Time) (int, error) {
	var total int
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	err := DB.Model(&Order{}).
		Select("COALESCE(SUM(money), 0)").
		Where("pay_status = ?", OrderPayStatusPaid).
		Where("pay_type = ?", OrderPayTypeAppleIAP).
		Where("pay_time >= ? AND pay_time <= ?", startOfDay, now).
		Scan(&total).Error
	return total, err
}

func UserHasPaidThirdPayOrder(userId int) (bool, error) {
	if userId <= 0 {
		return false, nil
	}
	var order Order
	err := DB.Select("id").
		Where("uid = ?", userId).
		Where("pay_status = ?", OrderPayStatusPaid).
		Where("pay_type IN ?", []string{OrderPayTypeXXPay, OrderPayTypeH5, OrderPayTypeSSPay}).
		First(&order).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return err == nil, err
}
