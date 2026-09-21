package controller

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"reflect"
	"regexp"
	"sync"
	"testing"
	"time"

	"just-vpn/middleware"
	"just-vpn/model"
	"just-vpn/pkg/mapping"
	appredis "just-vpn/pkg/redis"
	"just-vpn/pkg/setting"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	goredis "github.com/go-redis/redis"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestLuckyWheelPlayEligibility(t *testing.T) {
	for _, tc := range []struct {
		name, general, news string
		userType            int
		allowed             bool
	}{
		{"both off guest", "off", "off", 1, false},
		{"both off account", "off", "off", 2, false},
		{"general on guest", "on", "off", 1, true},
		{"general on account", "on", "off", 2, true},
		{"new user on guest", "off", "on", 1, true},
		{"new user on account", "off", "on", 2, false},
		{"both on guest", "on", "on", 1, true},
		{"both on account", "on", "on", 2, true},
		{"missing configuration", "", "", 1, false},
		{"invalid values", " ON ", "enabled", 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sqlDB, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer sqlDB.Close()
			now := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
			db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{
				NowFunc: func() time.Time { return now }, Logger: logger.Default.LogMode(logger.Silent),
			})
			if err != nil {
				t.Fatal(err)
			}
			oldDB := model.DB
			model.DB = db
			defer func() { model.DB = oldDB }()
			for _, config := range []struct{ code, value string }{
				{model.ConfigLuckyWheelGeneralEnabled, tc.general},
				{model.ConfigLuckyWheelNewUserEnabled, tc.news},
			} {
				rows := sqlmock.NewRows([]string{"code", "value"})
				if config.value != "" {
					rows.AddRow(config.code, config.value)
				}
				mock.ExpectQuery("SELECT .* FROM `config`").WithArgs(config.code, 1).WillReturnRows(rows)
			}
			client := goredis.NewClient(&goredis.Options{Addr: "unused:6379"})
			defer client.Close()
			oldRedis := appredis.Redis
			appredis.Redis = client
			defer func() { appredis.Redis = oldRedis }()
			reservations := 0
			client.WrapProcess(func(_ func(goredis.Cmder) error) func(goredis.Cmder) error {
				return func(cmd goredis.Cmder) error {
					if !tc.allowed {
						t.Fatal("ineligible user reached Redis")
					}
					if cmd.Name() != "set" || cmd.Args()[1] != "lucky_wheel:played:20260921:42" {
						t.Fatalf("unexpected command: %v", cmd.Args())
					}
					reservations++
					// 已有标记停止抽奖，验证配置放行后仍执行每日次数限制。
					*cmd.(*goredis.BoolCmd) = *goredis.NewBoolResult(false, nil)
					return nil
				}
			})
			router := gin.New()
			path := mapping.Endpoint("/api/v1/lucky_wheel_play")
			router.POST(path, func(c *gin.Context) {
				c.Set(middleware.ContextUserKey, model.User{
					BaseModel: model.BaseModel{Id: 42, CreateTime: now.Add(-48 * time.Hour)}, Type: tc.userType,
				})
				LuckyWheelPlayHandler(c)
			})
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest("POST", path, nil))
			var response map[string]interface{}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			wantMessage := "当前用户暂不可参与幸运转盘"
			wantReservations := 0
			if tc.allowed {
				wantMessage, wantReservations = "您今天已参与幸运转盘，请明天再试", 1
			}
			if recorder.Code != 200 || response["mid_call_midc_call_fix"] != float64(500) || response["mid_call_midm_call_fix"] != wantMessage {
				t.Fatalf("unexpected response: %s", recorder.Body.String())
			}
			if reservations != wantReservations {
				t.Fatalf("reservations = %d, want %d", reservations, wantReservations)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestLuckyWheelGetStatus(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		now                   time.Time
		key                   string
		played                int64
		news, general         string
		wantNews, wantGeneral string
		redisErr              error
	}{
		{"played before midnight", time.Date(2026, 9, 20, 15, 59, 59, 0, time.UTC), "lucky_wheel:played:20260920:42", 1, "on", "off", "on", "off", nil},
		{"new China day preserves whitespace", time.Date(2026, 9, 20, 16, 0, 0, 0, time.UTC), "lucky_wheel:played:20260921:42", 0, "off", " ON ", "off", " ON ", nil},
		{"arbitrary strings", time.Date(2026, 9, 20, 16, 0, 0, 0, time.UTC), "lucky_wheel:played:20260921:42", 0, "custom", "1", "custom", "1", nil},
		{"missing configs return empty", time.Date(2026, 9, 20, 16, 0, 0, 0, time.UTC), "lucky_wheel:played:20260921:42", 0, "", "", "", "", nil},
		{"redis failure", time.Date(2026, 9, 20, 16, 0, 0, 0, time.UTC), "lucky_wheel:played:20260921:42", 0, "", "", "", "", errors.New("redis unavailable")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sqlDB, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer sqlDB.Close()
			db, err := gorm.Open(mysql.New(mysql.Config{Conn: sqlDB, SkipInitializeWithVersion: true}), &gorm.Config{
				NowFunc: func() time.Time { return tc.now }, Logger: logger.Default.LogMode(logger.Silent),
			})
			if err != nil {
				t.Fatal(err)
			}
			oldDB := model.DB
			model.DB = db
			defer func() { model.DB = oldDB }()
			client := goredis.NewClient(&goredis.Options{Addr: "unused:6379"})
			defer client.Close()
			oldRedis := appredis.Redis
			appredis.Redis = client
			defer func() { appredis.Redis = oldRedis }()
			reads := 0
			client.WrapProcess(func(_ func(goredis.Cmder) error) func(goredis.Cmder) error {
				return func(cmd goredis.Cmder) error {
					if cmd.Name() != "exists" || len(cmd.Args()) != 2 || cmd.Args()[1] != tc.key {
						t.Fatalf("unexpected Redis operation: %v", cmd.Args())
					}
					reads++
					*cmd.(*goredis.IntCmd) = *goredis.NewIntResult(tc.played, tc.redisErr)
					return cmd.Err()
				}
			})
			if tc.redisErr == nil {
				for _, config := range []struct{ code, value string }{
					{model.ConfigLuckyWheelNewUserEnabled, tc.news},
					{model.ConfigLuckyWheelGeneralEnabled, tc.general},
				} {
					rows := sqlmock.NewRows([]string{"code", "value"})
					if config.value != "" {
						rows.AddRow(config.code, config.value)
					}
					mock.ExpectQuery("SELECT .* FROM `config`").WithArgs(config.code, 1).WillReturnRows(rows)
				}
			}
			router := gin.New()
			path := mapping.Endpoint("/api/v1/lucky_wheel_get_status")
			router.GET(path, func(c *gin.Context) {
				c.Set(middleware.ContextUserKey, model.User{BaseModel: model.BaseModel{Id: 42}})
				LuckyWheelGetStatusHandler(c)
			})
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, httptest.NewRequest("GET", path, nil))
			var response map[string]interface{}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != 200 {
				t.Fatalf("HTTP status = %d", recorder.Code)
			}
			if tc.redisErr != nil {
				if response["mid_call_midc_call_fix"] != float64(500) {
					t.Fatalf("response = %v", response)
				}
			} else {
				want := map[string]interface{}{
					"mid_call_midc_call_fix": float64(200),
					"mid_call_midm_call_fix": "success",
					"mid_call_rslt_call_fix": map[string]interface{}{
						"mid_call_tdpl_call_fix": tc.played == 1,
						"mid_call_nwen_call_fix": tc.wantNews,
						"mid_call_gnen_call_fix": tc.wantGeneral,
					},
				}
				if !reflect.DeepEqual(response, want) {
					t.Fatalf("response = %#v, want %#v", response, want)
				}
			}
			if reads != 1 {
				t.Fatalf("Redis reads = %d", reads)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

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
	for _, scenario := range []string{"active account", "expired guest", "missing VIP", "already claimed", "no record", "invalid seconds", "account failure", "sync failure", "status failure", "redis unavailable", "commit failure"} {
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
			cache := mockLuckyWheelClaimRedis(t, scenario == "redis unavailable")
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
			if scenario == "commit failure" {
				wantErr = "commit failed"
				mock.ExpectCommit().WillReturnError(errors.New(wantErr))
			} else if wantErr == "" {
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
			if scenario != "redis unavailable" {
				keepMarker := wantErr == "" || scenario == "already claimed"
				if (cache.marker != "") != keepMarker {
					t.Fatalf("marker retained = %v, want %v", cache.marker != "", keepMarker)
				}
				if keepMarker {
					// 后续并发请求均由 Redis 拦截，不得再开启数据库事务。
					var requests sync.WaitGroup
					for i := 0; i < 10; i++ {
						requests.Add(1)
						go func() {
							defer requests.Done()
							if err := claimLuckyWheel(42); err == nil || err.Error() != "lucky wheel reward claiming or claimed" {
								t.Errorf("repeat request error = %v", err)
							}
						}()
					}
					requests.Wait()
				} else if cache.releases != 1 {
					t.Fatalf("release count = %d, want 1", cache.releases)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

type luckyWheelClaimCache struct {
	mu       sync.Mutex
	marker   string
	releases int
}

func mockLuckyWheelClaimRedis(t *testing.T, unavailable bool) *luckyWheelClaimCache {
	t.Helper()
	state := &luckyWheelClaimCache{}
	client := goredis.NewClient(&goredis.Options{Addr: "unused:6379"})
	oldRedis := appredis.Redis
	appredis.Redis = client
	t.Cleanup(func() { appredis.Redis = oldRedis; _ = client.Close() })
	client.WrapProcess(func(_ func(goredis.Cmder) error) func(goredis.Cmder) error {
		return func(cmd goredis.Cmder) error {
			state.mu.Lock()
			defer state.mu.Unlock()
			args := cmd.Args()
			switch cmd.Name() {
			case "set":
				if len(args) != 6 || args[1] != "lucky_wheel:claimed:20260921:42" || args[3] != "ex" || args[4] != int64(86340) || args[5] != "nx" {
					t.Errorf("unexpected claim reservation: %v", args)
				}
				var err error
				if unavailable {
					err = errors.New("redis unavailable")
				}
				acquired := state.marker == "" && err == nil
				if acquired {
					state.marker = args[2].(string)
				}
				*cmd.(*goredis.BoolCmd) = *goredis.NewBoolResult(acquired, err)
			case "eval":
				if args[3] != "lucky_wheel:claimed:20260921:42" || args[4] != state.marker {
					t.Errorf("release does not own marker: %v", args)
				}
				state.marker = ""
				state.releases++
				*cmd.(*goredis.Cmd) = *goredis.NewCmdResult(int64(1), nil)
			default:
				t.Errorf("unexpected Redis command: %v", args)
			}
			return cmd.Err()
		}
	})
	return state
}
