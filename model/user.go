package model

import (
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

const (
	randomUserIDMin     = 100000000
	randomUserIDMax     = 999999999
	randomUserIDRetries = 20
	transferCodeLength  = 6
	transferCodeRetries = 20
	transferCodeChars   = "0123456789abcdefghijklmnopqrstuvwxyz"
)

// User 用户表
type User struct {
	BaseModel
	DeviceNo           string     `json:"device_no" gorm:"column:device_no;type:varchar(128);uniqueIndex:uk_user_device_no;comment:设备号"`
	TransferCode       string     `json:"transfer_code" gorm:"column:transfer_code;type:varchar(32);uniqueIndex:uk_user_transfer_code;comment:会员转移码"`
	Status             int        `json:"status" gorm:"column:status;type:int;comment:状态(0=封禁,1=正常)"`
	Type               int        `json:"type" gorm:"column:type;type:int;comment:账号类型(1=游客,2=账户)"`
	Username           string     `json:"username" gorm:"column:username;type:varchar(128);index:idx_username;comment:账户名"`
	VipTime            *time.Time `json:"vip_time" gorm:"column:vip_time;type:datetime;comment:会员到期时间"`
	InviterId          *int       `json:"inviter_id" gorm:"column:inviter_id;type:int;index:idx_inviter_id;comment:邀请人用户ID"`
	InvitedAt          *time.Time `json:"invited_at" gorm:"column:invited_at;type:datetime;comment:被邀请时间"`
	Platform           string     `json:"platform" gorm:"column:platform;type:varchar(32);comment:设备平台"`
	Version            string     `json:"version" gorm:"column:version;type:varchar(64);comment:版本号"`
	AppStoreRegion     string     `json:"app_store_region" gorm:"column:app_store_region;type:varchar(64);comment:苹果 App Store 地区"`
	Language           string     `json:"language" gorm:"column:language;type:varchar(16);comment:用户语言，zh-Hans简体中文、en英文"`
	LoginTimes         int        `json:"login_times" gorm:"column:login_times;type:int;comment:登录次数"`
	LastIp             string     `json:"last_ip" gorm:"column:last_ip;type:varchar(64);comment:最后登录IP"`
	LastIpRegion       string     `json:"last_ip_region" gorm:"column:last_ip_region;type:varchar(255);comment:最后登录地区"`
	LastLoginTime      *time.Time `json:"last_login_time" gorm:"column:last_login_time;type:datetime;comment:最后登录时间"`
	IsPay              int        `json:"is_pay" gorm:"column:is_pay;type:int;comment:是否支付(-1=否,1=是)"`
	PayTimes           int        `json:"pay_times" gorm:"column:pay_times;type:int;comment:支付次数"`
	PayAll             float64    `json:"pay_all" gorm:"column:pay_all;type:decimal(10,2);comment:总支付金额"`
	RegIp              string     `json:"reg_ip" gorm:"column:reg_ip;type:varchar(64);comment:注册IP"`
	RegIpRegion        string     `json:"reg_ip_region" gorm:"column:reg_ip_region;type:varchar(255);comment:注册IP区域"`
	Country            string     `json:"country" gorm:"column:country;type:varchar(64);comment:国家"`
	ShareCount         int        `json:"share_count" gorm:"column:share_count;type:int;comment:分享次数"`
	RewardTime         int        `json:"reward_time" gorm:"column:reward_time;type:int;comment:奖励时间，单位秒"`
	VpnFlow            int        `json:"vpn_flow" gorm:"column:vpn_flow;type:int;comment:VPN流量使用情况，单位K"`
	TodayVpnFlow       int        `json:"today_vpn_flow" gorm:"column:today_vpn_flow;type:int;comment:今日VPN使用流量情况，单位K"`
	IsReal             int        `json:"is_real" gorm:"column:is_real;type:int;comment:真实用户标记(-1=非真实,1=真实)"`
	IsTodayActive      int        `json:"is_today_active" gorm:"column:is_today_active;type:int;comment:今日是否活跃(1=是,-1=否)"`
	IsFlowClose        int        `json:"is_flow_close" gorm:"column:is_flow_close;type:int;comment:是否流量阈值封禁(1=正常,-1=流量封禁)"`
	DeviceStatus       int        `json:"device_status" gorm:"column:device_status;type:int;comment:设备状态(1=正常,-1=多次切换登录异常封禁)"`
	DeviceBanTime      *time.Time `json:"device_ban_time" gorm:"column:device_ban_time;type:datetime;comment:设备切换封禁时间"`
	MobileName         string     `json:"mobile_name" gorm:"column:mobile_name;type:varchar(128);comment:手机设备名称"`
	MobileVersion      string     `json:"mobile_version" gorm:"column:mobile_version;type:varchar(128);comment:手机软件版本"`
	MobileModelName    string     `json:"mobile_model_name" gorm:"column:mobile_model_name;type:varchar(128);comment:手机型号名称"`
	SourceChannel      string     `json:"source_channel" gorm:"column:source_channel;type:varchar(128);comment:来源渠道"`
	MigrationSource    *string    `json:"migration_source" gorm:"column:migration_source;type:varchar(128);index:idx_migration_source;comment:迁移来源项目名称"`
	RegisterCiphertext string     `json:"register_ciphertext" gorm:"column:register_ciphertext;type:varchar(255);index:idx_register_ciphertext;comment:注册密文"`
}

func (User) TableName() string { return "user" }

func GetUserByID(id int) (User, error) {
	var user User
	err := DB.Where("id = ?", id).Where("status = ?", UserStatusNormal).First(&user).Error
	return user, err
}

func GetUserByDeviceNo(deviceNo string) (User, error) {
	var user User
	err := DB.Where("device_no = ?", deviceNo).Where("status = ?", UserStatusNormal).Order("id desc").First(&user).Error
	return user, err
}

func GetUserByDeviceNoAnyStatus(deviceNo string) (User, error) {
	var user User
	err := DB.Where("device_no = ?", deviceNo).Order("id desc").First(&user).Error
	return user, err
}

func GetUserByUsername(username string) (User, error) {
	var user User
	err := DB.Where("username = ?", username).Where("status = ?", UserStatusNormal).Where("type = ?", 2).First(&user).Error
	return user, err
}

func GetUsersByUsername(username string) ([]User, error) {
	var users []User
	err := DB.Where("username = ?", username).Where("status = ?", UserStatusNormal).Where("type = ?", 2).Find(&users).Error
	return users, err
}

func CreateUser(user User) error {
	return CreateUserWithRandomID(DB, &user)
}

func CreateUserWithRandomID(db *gorm.DB, user *User) error {
	if db == nil {
		db = DB
	}
	if user.Id != 0 {
		return db.Create(user).Error
	}
	for i := 0; i < randomUserIDRetries; i++ {
		id, err := GenerateRandomUserID()
		if err != nil {
			return err
		}
		user.Id = id
		err = db.Create(user).Error
		if err == nil {
			return nil
		}
		if IsDuplicateEntryError(err) {
			user.Id = 0
			continue
		}
		return err
	}
	return gorm.ErrDuplicatedKey
}

func CreateUserWithRandomIDAndTransferCode(db *gorm.DB, user *User, prefix string) error {
	if db == nil {
		db = DB
	}
	for i := 0; i < transferCodeRetries; i++ {
		code, err := GenerateRandomTransferCode(prefix)
		if err != nil {
			return err
		}
		user.TransferCode = code
		if err := CreateUserWithRandomID(db, user); err != nil {
			if IsDuplicateEntryError(err) {
				user.Id = 0
				user.TransferCode = ""
				continue
			}
			return err
		}
		return nil
	}
	return gorm.ErrDuplicatedKey
}

func GenerateRandomUserID() (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(randomUserIDMax-randomUserIDMin+1))
	if err != nil {
		return 0, err
	}
	return randomUserIDMin + int(n.Int64()), nil
}

func GenerateRandomTransferCode(prefix string) (string, error) {
	prefix = strings.TrimSpace(strings.ToLower(prefix))
	if prefix == "" {
		prefix = "app"
	}
	code := make([]byte, transferCodeLength)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(transferCodeChars))))
		if err != nil {
			return "", err
		}
		code[i] = transferCodeChars[n.Int64()]
	}
	return prefix + string(code), nil
}

func EnsureUserTransferCode(db *gorm.DB, user *User, prefix string) error {
	if db == nil {
		db = DB
	}
	if user == nil || user.Id <= 0 || strings.TrimSpace(user.TransferCode) != "" {
		return nil
	}
	for i := 0; i < transferCodeRetries; i++ {
		code, err := GenerateRandomTransferCode(prefix)
		if err != nil {
			return err
		}
		result := db.Model(&User{}).
			Where("id = ?", user.Id).
			Where("(transfer_code = '' OR transfer_code IS NULL)").
			Update("transfer_code", code)
		if result.Error == nil && result.RowsAffected > 0 {
			user.TransferCode = code
			return nil
		}
		if result.Error == nil {
			var latest User
			if err := db.Select("transfer_code").Where("id = ?", user.Id).First(&latest).Error; err != nil {
				return err
			}
			if strings.TrimSpace(latest.TransferCode) != "" {
				user.TransferCode = latest.TransferCode
				return nil
			}
			continue
		}
		if IsDuplicateEntryError(result.Error) {
			continue
		}
		return result.Error
	}
	return gorm.ErrDuplicatedKey
}

func IsDuplicateEntryError(err error) bool {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}
	var mysqlErr *mysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}

func UpdateUserByID(id int, user User) error {
	return DB.Where("id = ?", id).Updates(&user).Error
}

func UpdateUserFieldsByID(id int, fields map[string]interface{}) error {
	return DB.Model(&User{}).Where("id = ?", id).Updates(fields).Error
}

func AddUserVpnFlow(id int, flow int) error {
	return DB.Model(&User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"vpn_flow":       gorm.Expr("vpn_flow + ?", flow),
		"today_vpn_flow": gorm.Expr("today_vpn_flow + ?", flow),
	}).Error
}
