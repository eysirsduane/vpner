package model

// PublicAppleId 公共 Apple ID 表
type PublicAppleId struct {
	BaseModel
	Appid  string `json:"appid" gorm:"column:appid;type:varchar(128);comment:Apple ID账号"`
	Pwd    string `json:"pwd" gorm:"column:pwd;type:varchar(128);comment:Apple ID密码"`
	Status int    `json:"status" gorm:"column:status;type:int;comment:状态(1=可用,0=禁用)"`
}

func (PublicAppleId) TableName() string { return "public_apple_id" }

func GetAvailablePublicAppleIds() ([]PublicAppleId, error) {
	var apples []PublicAppleId
	err := DB.Where("status = ?", 1).Order("id ASC").Find(&apples).Error
	return apples, err
}
