package model

import (
	"fmt"

	"just-vpn/pkg/setting"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init() error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Asia%%2FShanghai",
		setting.DatabaseConfig.User,
		setting.DatabaseConfig.Password,
		setting.DatabaseConfig.Host,
		setting.DatabaseConfig.DbName,
	)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxOpenConns(setting.DatabaseConfig.MaxOpenConns)
	sqlDB.SetMaxIdleConns(setting.DatabaseConfig.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(setting.DatabaseConfig.ConnMaxLifetime)

	DB = db
	return AutoMigrate()
}

func AutoMigrate() error {
	err := DB.AutoMigrate(
		&AppleNotification{},
		&Advert{},
		&Config{},
		&DailyStat{},
		&DelayedPopup{},
		&Error{},
		&InviteConfig{},
		&Node{},
		&NodeArea{},
		&NodeConnectHistory{},
		&NodeConnectLog{},
		&Order{},
		&Package{},
		&PayConfig{},
		&Popup{},
		&PublicAppleId{},
		&SystemNotice{},
		&SystemNoticeRead{},
		&TurnPayExposureLog{},
		&User{},
		&UserAccount{},
		&UserAdvert{},
		&UserBan{},
		&UserInviteCode{},
		&UserNotice{},
		&UserPopup{},
		&UserRecord{},
		&Version{},
		&ThirdPayCallback{},
	)
	if err != nil {
		return err
	}
	if err := EnsureNodeConnectLogIndexes(); err != nil {
		return err
	}
	if err := EnsureNodeConnectHistoryIndexes(); err != nil {
		return err
	}
	if err := EnsureLogCleanupIndexes(); err != nil {
		return err
	}
	if err := InitInviteConfigs(); err != nil {
		return err
	}
	if err := InitConfigs(); err != nil {
		return err
	}
	if err := InitPayConfigs(); err != nil {
		return err
	}
	if err := InitSeedData(); err != nil {
		return err
	}
	return InitConfigCaches()
}
