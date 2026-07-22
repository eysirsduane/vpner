package model

// UserInviteCode 用户邀请码表
type UserInviteCode struct {
	BaseModel
	UserId     int    `json:"user_id" gorm:"column:user_id;type:int;uniqueIndex:uniq_user_id;comment:用户ID"`
	InviteCode string `json:"invite_code" gorm:"column:invite_code;type:varchar(64);uniqueIndex:uniq_invite_code;comment:邀请码"`
	Status     int    `json:"status" gorm:"column:status;type:int;comment:状态(1=正常,0=禁用)"`
}

func (UserInviteCode) TableName() string { return "user_invite_code" }

func GetUserInviteCodeByUserID(userID int) (UserInviteCode, error) {
	var inviteCode UserInviteCode
	err := DB.Where("user_id = ?", userID).First(&inviteCode).Error
	return inviteCode, err
}

func GetUserInviteCodeByCode(code string) (UserInviteCode, error) {
	var inviteCode UserInviteCode
	err := DB.Where("invite_code = ?", code).Where("status = ?", 1).First(&inviteCode).Error
	return inviteCode, err
}
