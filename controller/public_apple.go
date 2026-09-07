package controller

import (
	"math/rand"
	"time"

	"just-vpn/model"
	"just-vpn/pkg/redis"

	"github.com/gin-gonic/gin"
)

const (
	publicAppleListCacheKey  = "PUBLIC_APPLE_ID_LIST"
	publicAppleIPCachePrefix = "PUBLIC_APPLE_ID_IP_"
	publicAppleReturnLimit   = 4
)

// PublicAppleIdResponse 公共 Apple ID 信息
type PublicAppleIdResponse struct {
	Id    int    `json:"id" example:"1"`                    // 记录ID
	Appid string `json:"appid" example:"apple@example.com"` // Apple ID账号
	Pwd   string `json:"pwd" example:"apple-password"`      // Apple ID密码
}

// PublicAppleIdHandler 获取公共 Apple ID
// @Summary 获取公共 Apple ID
// @Description 获取可用的公共 Apple ID 列表，每个 IP 24 小时内返回同一组账号
// @Tags 支付
// @Produce json
// @Success 200 {object} Response{result=[]PublicAppleIdResponse}
// @Router /public_apple_id [get]
func PublicAppleIdHandler(c *gin.Context) {
	ip := c.ClientIP()
	if ip == "" {
		JsonReturn(c, CodeError, "ip error", nil)
		return
	}

	ipCacheKey := publicAppleIPCachePrefix + ip
	var cached []PublicAppleIdResponse
	if ok, err := redis.Get(ipCacheKey, &cached); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	} else if ok {
		JsonReturn(c, CodeSuccess, "success", cached)
		return
	}

	apples, err := availablePublicAppleIds()
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	result := pickPublicAppleIds(apples, publicAppleReturnLimit)
	if err := redis.Set(ipCacheKey, result, 24*time.Hour); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	JsonReturn(c, CodeSuccess, "success", result)
}

func availablePublicAppleIds() ([]model.PublicAppleId, error) {
	var apples []model.PublicAppleId
	if ok, err := redis.Get(publicAppleListCacheKey, &apples); err != nil {
		return nil, err
	} else if ok {
		return apples, nil
	}

	apples, err := model.GetAvailablePublicAppleIds()
	if err != nil {
		return nil, err
	}
	if err := redis.SetDefault(publicAppleListCacheKey, apples); err != nil {
		return nil, err
	}
	return apples, nil
}

func pickPublicAppleIds(apples []model.PublicAppleId, limit int) []PublicAppleIdResponse {
	if len(apples) == 0 || limit <= 0 {
		return []PublicAppleIdResponse{}
	}
	indexes := rand.New(rand.NewSource(time.Now().UnixNano())).Perm(len(apples))
	if len(indexes) > limit {
		indexes = indexes[:limit]
	}

	result := make([]PublicAppleIdResponse, 0, len(indexes))
	for _, index := range indexes {
		apple := apples[index]
		result = append(result, PublicAppleIdResponse{
			Id:    apple.Id,
			Appid: apple.Appid,
			Pwd:   apple.Pwd,
		})
	}
	return result
}
