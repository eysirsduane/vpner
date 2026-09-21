package model

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	appredis "just-vpn/pkg/redis"
	"just-vpn/pkg/setting"

	"github.com/DATA-DOG/go-sqlmock"
	goredis "github.com/go-redis/redis"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPlayLuckyWheelRedisRoundLimit(t *testing.T) {
	oldCache := systemConfigCache
	systemConfigCache = &configCacheStore{loaded: true, values: map[string]string{ConfigLuckyWheelRoundTime: "3600"}}
	defer func() { systemConfigCache = oldCache }()
	for _, scenario := range []string{"concurrent requests", "insert failure then retry", "redis unavailable", "empty pool"} {
		t.Run(scenario, func(t *testing.T) {
			sqlDB, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer sqlDB.Close()
			// 验证按配置冷却 3600 秒，而不是到次日零点。
			now := time.Date(2026, 9, 20, 16, 1, 0, 0, time.UTC)
			db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{
				NowFunc: func() time.Time { return now }, Logger: logger.Default.LogMode(logger.Silent),
			})
			if err != nil {
				t.Fatal(err)
			}
			client := goredis.NewClient(&goredis.Options{Addr: "unused:6379"})
			defer client.Close()
			oldDB, oldRedis := DB, appredis.Redis
			DB, appredis.Redis = db, client
			defer func() { DB, appredis.Redis = oldDB, oldRedis }()
			pool := []LuckyWheel{{BaseModel: BaseModel{Id: 1}, Level: 1, Seconds: 600, Title: "10分钟", Desc: "免费会员", Remark: "高速会员专属权益"}}
			if scenario == "empty pool" {
				pool = nil
			}
			poolJSON, _ := json.Marshal(pool)
			var mu sync.Mutex
			marker := ""
			released := 0
			client.WrapProcess(func(_ func(goredis.Cmder) error) func(goredis.Cmder) error {
				return func(cmd goredis.Cmder) error {
					mu.Lock()
					defer mu.Unlock()
					args := cmd.Args()
					switch cmd.Name() {
					case "set":
						if len(args) != 6 || args[1] != "lucky_wheel:played:42" || args[3] != "ex" || args[4] != int64(3600) || args[5] != "nx" {
							t.Errorf("unexpected round reservation: %v", args)
						}
						var setErr error
						if scenario == "redis unavailable" {
							setErr = errors.New("redis unavailable")
						}
						acquired := marker == "" && setErr == nil
						if acquired {
							marker = args[2].(string)
						}
						*cmd.(*goredis.BoolCmd) = *goredis.NewBoolResult(acquired, setErr)
					case "exists":
						if args[1] != luckyWheelListCacheKey {
							t.Errorf("unexpected cache key: %v", args)
						}
						*cmd.(*goredis.IntCmd) = *goredis.NewIntResult(1, nil)
					case "get":
						*cmd.(*goredis.StringCmd) = *goredis.NewStringResult(string(poolJSON), nil)
					case "eval":
						if args[3] != "lucky_wheel:played:42" || args[4] != marker {
							t.Errorf("release does not own marker: %v", args)
						}
						marker = ""
						released++
						*cmd.(*goredis.Cmd) = *goredis.NewCmdResult(int64(1), nil)
					default:
						t.Errorf("unexpected Redis command: %v", args)
					}
					return cmd.Err()
				}
			})
			insertSQL := "INSERT INTO `lucky_wheel_play_record`"
			if scenario == "insert failure then retry" {
				mock.ExpectExec(insertSQL).WillReturnError(errors.New("insert failed"))
				if _, err := PlayLuckyWheel(42); err == nil || !strings.Contains(err.Error(), "insert failed") {
					t.Fatalf("error = %v", err)
				}
				if marker != "" || released != 1 {
					t.Fatal("failed insert retained reservation")
				}
			}
			switch scenario {
			case "concurrent requests", "insert failure then retry":
				// 不允许 SELECT、BEGIN 或第二条 INSERT；同时校验秒数快照和未领取状态。
				mock.ExpectExec(insertSQL).
					WithArgs(now.In(setting.ChinaLocation), now, nil, 42, 600, 0, "10分钟", "免费会员", "高速会员专属权益").
					WillReturnResult(sqlmock.NewResult(1, 1))
				results := make(chan error, 20)
				for i := 0; i < 20; i++ {
					go func() { _, err := PlayLuckyWheel(42); results <- err }()
				}
				successes := 0
				for i := 0; i < 20; i++ {
					err := <-results
					if err == nil {
						successes++
					} else if !errors.Is(err, ErrLuckyWheelAlreadyPlayed) {
						t.Errorf("unexpected error: %v", err)
					}
				}
				if successes != 1 || marker == "" {
					t.Fatalf("successes = %d, marker = %q", successes, marker)
				}
			case "redis unavailable":
				if _, err := PlayLuckyWheel(42); err == nil || err.Error() != "redis unavailable" {
					t.Fatalf("error = %v", err)
				}
				if released != 0 {
					t.Fatal("released a reservation that was never acquired")
				}
			case "empty pool":
				if _, err := PlayLuckyWheel(42); !errors.Is(err, gorm.ErrRecordNotFound) {
					t.Fatalf("error = %v", err)
				}
				if marker != "" || released != 1 {
					t.Fatal("empty pool retained reservation")
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLuckyWheelRoundDuration(t *testing.T) {
	oldCache := systemConfigCache
	defer func() { systemConfigCache = oldCache }()
	for _, tc := range []struct {
		value string
		want  time.Duration
	}{
		{"3600", time.Hour}, {" 60 ", time.Minute}, {"172800", 48 * time.Hour},
		{"", 24 * time.Hour}, {"0", 24 * time.Hour}, {"-1", 24 * time.Hour},
		{"invalid", 24 * time.Hour}, {"9223372036854775807", 24 * time.Hour},
	} {
		t.Run(tc.value, func(t *testing.T) {
			systemConfigCache = &configCacheStore{loaded: true, values: map[string]string{ConfigLuckyWheelRoundTime: tc.value}}
			if got := LuckyWheelRoundDuration(); got != tc.want {
				t.Fatalf("duration = %v, want %v", got, tc.want)
			}
		})
	}
}
