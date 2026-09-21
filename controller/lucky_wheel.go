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
	TodayPlayed         bool   `json:"today_played" example:"false"`       // 北京时间当天是否已有参与标记
	NewsEnabled         string `json:"news_enabled" example:"off"`         // 新用户幸运转盘开关配置原值
	GeneralEnabled      string `json:"general_enabled" example:"off"`      // 通用幸运转盘开关配置原值
	NewsCloseEnabled    string `json:"news_close_enabled" example:"on"`    // 新用户幸运转盘界面关闭开关配置原值
	GeneralCloseEnabled string `json:"general_close_enabled" example:"on"` // 通用幸运转盘界面关闭开关配置原值
}

// LuckyWheelGetStatusHandler 获取当前用户的幸运转盘状态。
// @Summary 获取幸运转盘状态
// @Description 返回北京时间当天的Redis参与标记today_played，以及news_enabled、general_enabled、news_close_enabled、general_close_enabled四个开关配置。配置字符串原样返回，缺失、为空或读取失败时返回空字符串；查询沿用配置缓存，不修改参与标记或TTL。Redis读取失败返回业务状态码500。
// @Tags 活动
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
		TodayPlayed:         played,
		NewsEnabled:         model.ConfigValue(model.ConfigLuckyWheelNewUserEnabled, ""),
		GeneralEnabled:      model.ConfigValue(model.ConfigLuckyWheelGeneralEnabled, ""),
		NewsCloseEnabled:    model.ConfigValue(model.ConfigLuckyWheelNewUserCloseEnabled, ""),
		GeneralCloseEnabled: model.ConfigValue(model.ConfigLuckyWheelGeneralCloseEnabled, ""),
	})
}

// LuckyWheelPlayHandler 随机获取幸运转盘奖品。
// @Summary 随机获取幸运转盘奖品
// @Description 通用开关lucky_wheel.general_enabled为on，或新用户开关lucky_wheel.new_user_enabled为on且user.type=1，满足任一条件即可参与，不限制注册时长。
// @Tags 活动
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
// @Description 用户领取当天的中奖奖品
// @Tags 活动
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
// @Description 随机生成30条中奖记录，用户ID为随机数字前后拼接的字符串，长度不固定。
// @Tags 活动
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
