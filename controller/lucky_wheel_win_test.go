package controller

import (
	"database/sql/driver"
	"errors"
	"regexp"
	"testing"
	"time"

	"just-vpn/model"
	"just-vpn/pkg/setting"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type luckyWheelVIPTime struct {
	base    *time.Time
	seconds int
	before  time.Time
}

func (m luckyWheelVIPTime) Match(value driver.Value) bool {
	got, ok := value.(time.Time)
	if !ok {
		return false
	}
	if m.base != nil && m.base.After(m.before) {
		return got.Equal(m.base.Add(time.Duration(m.seconds) * time.Second).Truncate(time.Second))
	}
	return !got.Before(m.before.Add(time.Duration(m.seconds)*time.Second).Truncate(time.Second)) &&
		!got.After(chinaNow().Add(time.Duration(m.seconds)*time.Second))
}

func TestClaimLuckyWheel(t *testing.T) {
	for _, scenario := range []string{"active account", "expired guest", "missing VIP", "already claimed", "no record", "invalid seconds", "account failure", "sync failure", "status failure"} {
		t.Run(scenario, func(t *testing.T) {
			sqlDB, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer sqlDB.Close()
			// UTC 的前一天 16 点后已是中国时区的次日。
			fixedNow := time.Date(2026, 9, 20, 16, 1, 0, 0, time.UTC)
			db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{
				NowFunc: func() time.Time { return fixedNow }, Logger: logger.Default.LogMode(logger.Silent),
			})
			if err != nil {
				t.Fatal(err)
			}
			oldDB := model.DB
			model.DB = db
			defer func() { model.DB = oldDB }()
			mock.ExpectBegin()
			dayStart := time.Date(2026, 9, 21, 0, 0, 0, 0, setting.ChinaLocation)
			rows := sqlmock.NewRows([]string{"id", "user_id", "vip_secs", "status"})
			status, seconds := 0, 600
			if scenario == "already claimed" {
				status = 1
			}
			if scenario == "invalid seconds" {
				seconds = 0
			}
			if scenario != "no record" {
				rows.AddRow(7, 42, seconds, status)
			}
			mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `lucky_wheel_play_record` WHERE user_id = ? AND create_time >= ? AND create_time < ? ORDER BY `lucky_wheel_play_record`.`id` LIMIT ? FOR UPDATE")).
				WithArgs(42, dayStart, dayStart.AddDate(0, 0, 1), 1).WillReturnRows(rows)
			wantErr := ""
			switch scenario {
			case "already claimed":
				wantErr = "lucky wheel reward already claimed"
			case "no record":
				wantErr = "lucky wheel reward not found today"
			case "invalid seconds":
				wantErr = "lucky wheel reward duration invalid"
			default:
				before := chinaNow()
				vip := before.Add(24 * time.Hour)
				username, userType := "wheel@example.com", 2
				var storedVIP interface{} = vip
				var base *time.Time = &vip
				if scenario == "expired guest" || scenario == "missing VIP" {
					username, userType = "", 1
					vip = before.Add(-time.Hour)
					storedVIP = vip
					if scenario == "missing VIP" {
						storedVIP, base = nil, nil
					}
				}
				mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `user` WHERE id = ? AND status = ? ORDER BY `user`.`id` LIMIT ? FOR UPDATE")).
					WithArgs(42, model.UserStatusNormal, 1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "username", "type", "status", "vip_time"}).AddRow(42, username, userType, model.UserStatusNormal, storedVIP))
				vipArg := luckyWheelVIPTime{base: base, seconds: 600, before: before}
				// 精确匹配 UPDATE 列，确保没有修改 update_time 或任何支付字段。
				mock.ExpectExec(regexp.QuoteMeta("UPDATE `user` SET `vip_time`=? WHERE id = ?")).
					WithArgs(vipArg, 42).WillReturnResult(sqlmock.NewResult(0, 1))
				if username != "" {
					update := mock.ExpectExec(regexp.QuoteMeta("UPDATE `user_account` SET `vip_time`=? WHERE username = ?")).WithArgs(vipArg, username)
					if scenario == "account failure" {
						wantErr = "account update failed"
						update.WillReturnError(errors.New(wantErr))
					} else {
						update.WillReturnResult(sqlmock.NewResult(0, 1))
						sync := mock.ExpectExec(regexp.QuoteMeta("UPDATE `user` SET `vip_time`=? WHERE username = ? AND type = ? AND status = ?")).WithArgs(vipArg, username, 2, model.UserStatusNormal)
						if scenario == "sync failure" {
							wantErr = "sync failed"
							sync.WillReturnError(errors.New(wantErr))
						} else {
							sync.WillReturnResult(sqlmock.NewResult(0, 2))
						}
					}
				}
				if wantErr == "" {
					update := mock.ExpectExec(regexp.QuoteMeta("UPDATE `lucky_wheel_play_record` SET `status`=?,`update_time`=? WHERE id = ? AND status = ?")).WithArgs(1, fixedNow, 7, 0)
					if scenario == "status failure" {
						wantErr = "status update failed"
						update.WillReturnError(errors.New(wantErr))
					} else {
						update.WillReturnResult(sqlmock.NewResult(0, 1))
					}
				}
			}
			if wantErr == "" {
				mock.ExpectCommit()
			} else {
				mock.ExpectRollback()
			}
			err = claimLuckyWheel(42)
			if wantErr == "" && err != nil {
				t.Fatal(err)
			}
			if wantErr != "" && (err == nil || err.Error() != wantErr) {
				t.Fatalf("error = %v, want %q", err, wantErr)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
