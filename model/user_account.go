package model

import "time"

// UserAccount 用户账号表
type UserAccount struct {
	BaseModel
	Username        string     `json:"username" gorm:"column:username;type:varchar(128);index:idx_username;comment:账户名"`
	Password        string     `json:"password" gorm:"column:password;type:varchar(128);comment:密码"`
	VipTime         *time.Time `json:"vip_time" gorm:"column:vip_time;type:datetime;comment:会员到期时间"`
	MigrationSource *string    `json:"migration_source" gorm:"column:migration_source;type:varchar(128);index:idx_migration_source;comment:迁移来源项目名称"`
}

func (UserAccount) TableName() string { return "user_account" }

func GetUserAccountByUsername(username string) (UserAccount, error) {
	var account UserAccount
	err := DB.Where("username = ?", username).First(&account).Error
	return account, err
}

func CreateUserAccount(account UserAccount) error {
	return DB.Create(&account).Error
}

func UpdateUserAccountByUsername(username string, fields map[string]interface{}) error {
	return DB.Model(&UserAccount{}).Where("username = ?", username).Updates(fields).Error
}
