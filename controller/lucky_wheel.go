package controller

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"just-vpn/middleware"
	"just-vpn/model"
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

// LuckyWheelPlayHandler 随机获取幸运转盘奖品。
// @Summary 随机获取幸运转盘奖品
// @Description 仅限注册24小时内的新用户调用，使用Redis SET NX原子占用北京时间当天参与次数，标记于次日零点过期；不符合条件、当天已有标记或Redis不可用时返回业务状态码500。从lucky_wheel表中等概率随机读取一条未删除的记录，保存user_id、title、desc、remark、vip_secs及未领取状态至lucky_wheel_play_record表后返回level、title、desc、remark。抽奖或写入失败时释放本次占位。不查询数据库参与次数，不使用显式事务或GORM默认写入事务，不修改用户会员时长。Redis标记丢失后无法保证每日防重。
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
	if !isLuckyWheelNewUser(user.CreateTime, chinaNow()) {
		JsonReturn(c, CodeError, "lucky wheel is only available to new users", nil)
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

func isLuckyWheelNewUser(createTime time.Time, now time.Time) bool {
	return !createTime.IsZero() && !createTime.After(now) && now.Sub(createTime) <= 24*time.Hour
}

// LuckyWheelWinHandler 领取当天幸运转盘奖励。
// @Summary 领取幸运转盘奖励
// @Description 根据北京时间当天的参与记录领取会员时长，无需请求参数。同一记录只能领取一次；仅更新用户、账号及同账号正常用户的vip_time，领取状态与会员时间在同一事务中保存。
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
	return model.DB.Transaction(func(tx *gorm.DB) error {
		var record model.LuckyWhellPlayRecord
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
		result := tx.Model(&model.LuckyWhellPlayRecord{}).
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
