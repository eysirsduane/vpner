package controller

import (
	"fmt"
	"math/rand/v2"
	"time"

	"just-vpn/middleware"
	"just-vpn/model"

	"github.com/gin-gonic/gin"
)

type LuckyWheelPlayResponse struct {
	Level  int    `json:"level" example:"1"`
	Title  string `json:"title" example:"10分钟免费会员"`
	Remark string `json:"remark" example:"高速会员专属权益"`
}

// LuckyWheelPlayHandler 随机获取幸运转盘奖品。
// @Summary 随机获取幸运转盘奖品
// @Description 仅限注册24小时内的新用户调用，以服务器时间和用户创建时间判断；每个用户每天仅可参与一次，按北京时间零点重置。不符合条件或当天已参与返回业务状态码500。从lucky_wheel表中等概率随机读取一条未删除的记录，保存user_id、title、desc、remark至lucky_wheel_play_record表后返回level、title、remark。校验、抽奖和记录写入在同一事务中完成，不修改用户会员时长。
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
		Remark: prize.Remark,
	})
}

func isLuckyWheelNewUser(createTime time.Time, now time.Time) bool {
	return !createTime.IsZero() && !createTime.After(now) && now.Sub(createTime) <= 24*time.Hour
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
