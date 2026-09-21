package controller

import (
	cryptorand "crypto/rand"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"just-vpn/middleware"
	"just-vpn/model"
	"just-vpn/pkg/redis"
	"just-vpn/pkg/setting"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LuckyWheelPlayResponse struct {
	Level  int    `json:"level" example:"1"`
	Title  string `json:"title" example:"10分钟免费会员"`
	Desc   string `json:"desc" example:"免费会员"`
	Remark string `json:"remark" example:"高速会员专属权益"`
}

type LuckyWheelStatusResponse struct {
	TodayPlayed    bool   `json:"today_played" example:"false"`
	NewsEnabled    string `json:"news_enabled" example:"off"`
	GeneralEnabled string `json:"general_enabled" example:"off"`
}

// LuckyWheelGetStatusHandler 获取当前用户的幸运转盘状态。
// @Summary 获取幸运转盘状态
// @Description 按北京时间当天的Redis参与标记返回today_played布尔值；news_enabled与general_enabled原样返回系统配置字符串，不转换大小写、不去除空白、不限制取值，配置缺失或读取失败返回空字符串。沿用系统配置缓存。查询不会写入标记或延长TTL；Redis读取失败返回业务状态码500。
// @Tags 系统
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response{result=LuckyWheelStatusResponse}
// @Router /lucky_wheel_get_status [get]
func LuckyWheelGetStatusHandler(c *gin.Context) {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	played, err := model.LuckyWheelPlayedToday(user.Id)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	JsonReturn(c, CodeSuccess, "success", LuckyWheelStatusResponse{
		TodayPlayed:    played,
		NewsEnabled:    model.ConfigValue(model.ConfigLuckyWheelNewUserEnabled, ""),
		GeneralEnabled: model.ConfigValue(model.ConfigLuckyWheelGeneralEnabled, ""),
	})
}

// LuckyWheelPlayHandler 随机获取幸运转盘奖品。
// @Summary 随机获取幸运转盘奖品
// @Description 通用开关lucky_wheel.general_enabled为on，或新用户开关lucky_wheel.new_user_enabled为on且user.type=1，满足任一条件即可参与，不限制注册时长。使用Redis SET NX原子占用北京时间当天参与次数，标记于次日零点过期；不符合条件、当天已有标记或Redis不可用时返回业务状态码500。从lucky_wheel表中等概率随机读取一条未删除的记录，保存user_id、title、desc、remark、vip_secs及未领取状态至lucky_wheel_play_record表后返回level、title、desc、remark。抽奖或写入失败时释放本次占位。不查询数据库参与次数，不使用显式事务或GORM默认写入事务，不修改用户会员时长。Redis标记丢失后无法保证每日防重。
// @Tags 系统
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response{result=LuckyWheelPlayResponse}
// @Router /lucky_wheel_play [post]
func LuckyWheelPlayHandler(c *gin.Context) {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	generalEnabled := model.ConfigValue(model.ConfigLuckyWheelGeneralEnabled, "off") == "on"
	newUserEnabled := model.ConfigValue(model.ConfigLuckyWheelNewUserEnabled, "off") == "on"
	if !generalEnabled && !(newUserEnabled && user.Type == 1) {
		JsonReturn(c, CodeError, "lucky wheel is not available for this user", nil)
		return
	}
	prize, err := model.PlayLuckyWheel(user.Id)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	JsonReturn(c, CodeSuccess, "success", LuckyWheelPlayResponse{
		Level:  prize.Level,
		Title:  prize.Title,
		Desc:   prize.Desc,
		Remark: prize.Remark,
	})
}

// LuckyWheelGetRewardHandler 领取当天幸运转盘奖励。
// @Summary 领取幸运转盘奖励
// @Description 根据北京时间当天的参与记录领取会员时长，无需请求参数。Redis SET NX提前拦截重复领奖，标记于次日零点过期，失败释放本次标记；Redis不可用时回退数据库。同一记录只能领取一次，保留数据库状态校验和行锁；仅更新用户、账号及同账号正常用户的vip_time，领取状态与会员时间在同一事务中保存。
// @Tags 系统
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response
// @Router /lucky_wheel_get_reward [post]
func LuckyWheelGetRewardHandler(c *gin.Context) {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	if err := claimLuckyWheel(user.Id); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	JsonReturn(c, CodeSuccess, "success", nil)
}

func claimLuckyWheel(userID int) error {
	now := model.DB.NowFunc().In(setting.ChinaLocation)
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, setting.ChinaLocation)
	dayEnd := dayStart.AddDate(0, 0, 1)
	client := redis.Redis
	key := fmt.Sprintf("lucky_wheel:claimed:%s:%d", now.Format("20060102"), userID)
	token := cryptorand.Text()
	reserved, keepMarker := false, false
	if client != nil {
		acquired, err := client.SetNX(key, token, dayEnd.Sub(now)).Result()
		if err == nil {
			if !acquired {
				return errors.New("lucky wheel reward claiming or claimed")
			}
			reserved = true
		}
		// Redis 不可用时仍由数据库行锁与领取状态保证只发奖一次。
	}
	defer func() {
		if reserved && !keepMarker {
			// 校验请求标识后删除，避免误删其他请求的占位。
			_ = client.Eval(`if redis.call("GET", KEYS[1]) == ARGV[1] then return redis.call("DEL", KEYS[1]) end return 0`, []string{key}, token).Err()
		}
	}()
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		var record model.LuckyWheelPlayRecord
		// 不过滤 status，确保重复调用始终检查同一条记录。
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ? AND create_time >= ? AND create_time < ?", userID, dayStart, dayEnd).
			First(&record).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("lucky wheel reward not found today")
			}
			return err
		}
		if record.Status == model.LuckyWheelClaimed {
			keepMarker = true // 缓存数据库已确认的领取状态。
			return errors.New("lucky wheel reward already claimed")
		}
		if record.Status != model.LuckyWheelUnclaimed || record.DeleteTime != nil || record.VipSeconds <= 0 {
			return errors.New("lucky wheel reward duration invalid")
		}
		var user model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("id = ? AND status = ?", userID, model.UserStatusNormal).First(&user).Error; err != nil {
			return err
		}
		vipTime := addVipSeconds(user.VipTime, record.VipSeconds)
		// UpdateColumn 跳过自动更新时间，用户和账号表仅修改 vip_time。
		if err := tx.Model(&model.User{}).Where("id = ?", user.Id).UpdateColumn("vip_time", vipTime).Error; err != nil {
			return err
		}
		if user.Username != "" && user.Type == 2 {
			if err := tx.Model(&model.UserAccount{}).Where("username = ?", user.Username).
				UpdateColumn("vip_time", vipTime).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.User{}).
				Where("username = ? AND type = ? AND status = ?", user.Username, 2, model.UserStatusNormal).
				UpdateColumn("vip_time", vipTime).Error; err != nil {
				return err
			}
		}
		result := tx.Model(&model.LuckyWheelPlayRecord{}).
			Where("id = ? AND status = ?", record.Id, model.LuckyWheelUnclaimed).
			Update("status", model.LuckyWheelClaimed)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return errors.New("lucky wheel reward already claimed")
		}
		return nil
	})
	if err == nil {
		keepMarker = true
	}
	return err
}

// LuckyWheelWinnerResponse 随机生成的用户中奖记录。
type LuckyWheelWinnerResponse struct {
	UserID string `json:"user_id" example:"123...4"`
	Title  string `json:"title" example:"10分钟"`
	Desc   string `json:"desc" example:"免费会员"`
	Remark string `json:"remark" example:"高速会员专属权益"`
}

// LuckyWheelWinnersHandler 获取用户中奖记录。
// @Summary 获取用户中奖记录
// @Description 读取lucky_wheel表中全部未删除的数据，等概率有放回抽取并生成30条展示记录，title、desc、remark来自同一条转盘记录。user_id由100至999的随机数、三个英文句点和0至9的随机数组成。无可用记录时返回业务状态码500，不保存生成的中奖记录。
// @Tags 系统
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response{result=[]LuckyWheelWinnerResponse}
// @Router /lucky_wheel_winners [get]
func LuckyWheelWinnersHandler(c *gin.Context) {
	luckyWheels, err := model.GetAvailableLuckyWheels()
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	if len(luckyWheels) == 0 {
		JsonReturn(c, CodeError, "record not found", nil)
		return
	}
	result := make([]LuckyWheelWinnerResponse, 30)
	for i := range result {
		luckyWheel := luckyWheels[rand.IntN(len(luckyWheels))]
		result[i] = LuckyWheelWinnerResponse{
			UserID: fmt.Sprintf("%d...%d", rand.IntN(900)+100, rand.IntN(10)),
			Title:  luckyWheel.Title,
			Desc:   luckyWheel.Desc,
			Remark: luckyWheel.Remark,
		}
	}

	JsonReturn(c, CodeSuccess, "success", result)
}
