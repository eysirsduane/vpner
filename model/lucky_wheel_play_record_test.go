package model

import (
	"errors"
	"regexp"
	"testing"
	"time"

	"just-vpn/pkg/setting"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPlayLuckyWheelTransaction(t *testing.T) {
	originalLocal := time.Local
	time.Local = time.UTC
	t.Cleanup(func() { time.Local = originalLocal })
	failure := errors.New("database operation failed")
	beforeMidnight := time.Date(2026, 9, 20, 23, 59, 59, 0, setting.ChinaLocation)
	for _, test := range []struct {
		name  string
		stage string
		now   time.Time
		want  error
	}{
		{"first play stores prize", "success", beforeMidnight, nil},
		{"new day uses new quota", "success", beforeMidnight.Add(time.Second), nil},
		{"UTC clock before China midnight", "success", beforeMidnight.UTC(), nil},
		{"UTC clock at China midnight", "success", beforeMidnight.Add(time.Second).UTC(), nil},
		{"already played", "duplicate", beforeMidnight, ErrLuckyWheelAlreadyPlayed},
		{"user lock failure", "lock", beforeMidnight, failure},
		{"quota lookup failure", "count", beforeMidnight, failure},
		{"no available prize", "empty", beforeMidnight, gorm.ErrRecordNotFound},
		{"prize query failure", "prize", beforeMidnight, failure},
		{"record insert failure", "insert", beforeMidnight, failure},
		{"commit failure", "commit", beforeMidnight, failure},
	} {
		t.Run(test.name, func(t *testing.T) {
			sqlDB, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = sqlDB.Close() })
			db, err := gorm.Open(mysql.New(mysql.Config{
				Conn: sqlDB, SkipInitializeWithVersion: true,
			}), &gorm.Config{
				NowFunc: func() time.Time { return test.now },
				Logger:  logger.Default.LogMode(logger.Silent),
			})
			if err != nil {
				t.Fatal(err)
			}
			originalDB := DB
			DB = db
			t.Cleanup(func() { DB = originalDB })

			const userID = 123456789
			mock.ExpectBegin()
			lock := mock.ExpectQuery(regexp.QuoteMeta("SELECT `id` FROM `user` WHERE id = ? LIMIT ? FOR UPDATE")).
				WithArgs(userID, 1)
			if test.stage == "lock" {
				lock.WillReturnError(failure)
			} else {
				lock.WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(userID))
				start := time.Date(2026, 9, test.now.In(setting.ChinaLocation).Day(), 0, 0, 0, 0, setting.ChinaLocation)
				count := mock.ExpectQuery(regexp.QuoteMeta("SELECT count(*) FROM `lucky_wheel_play_record` WHERE user_id = ? AND create_time >= ? AND create_time < ?")).
					WithArgs(userID, start, start.AddDate(0, 0, 1))
				switch test.stage {
				case "count":
					count.WillReturnError(failure)
				case "duplicate":
					count.WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
				default:
					count.WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
					prize := mock.ExpectQuery(regexp.QuoteMeta("SELECT * FROM `lucky_wheel` WHERE delete_time IS NULL ORDER BY RAND() LIMIT ?")).WithArgs(1)
					rows := sqlmock.NewRows([]string{"id", "level", "title", "desc", "remark"})
					switch test.stage {
					case "empty":
						prize.WillReturnRows(rows)
					case "prize":
						prize.WillReturnError(failure)
					default:
						prize.WillReturnRows(rows.AddRow(7, 3, "30分钟", "免费会员", "高速会员专属权益"))
						insert := mock.ExpectExec("^INSERT INTO `lucky_wheel_play_record`").
							WithArgs(test.now.In(setting.ChinaLocation), test.now, nil, userID, "30分钟", "免费会员", "高速会员专属权益")
						if test.stage == "insert" {
							insert.WillReturnError(failure)
						} else {
							insert.WillReturnResult(sqlmock.NewResult(11, 1))
						}
					}
				}
			}
			switch test.stage {
			case "success":
				mock.ExpectCommit()
			case "commit":
				mock.ExpectCommit().WillReturnError(failure)
			default:
				mock.ExpectRollback()
			}

			prize, err := PlayLuckyWheel(userID)
			if !errors.Is(err, test.want) {
				t.Fatalf("PlayLuckyWheel() error = %v, want %v", err, test.want)
			}
			if test.want == nil {
				if prize.Level != 3 || prize.Title != "30分钟" || prize.Desc != "免费会员" || prize.Remark != "高速会员专属权益" {
					t.Fatalf("unexpected prize: %#v", prize)
				}
			} else if prize != (LuckyWheel{}) {
				t.Fatalf("failed transaction returned prize: %#v", prize)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
