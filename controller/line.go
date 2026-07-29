package controller

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"just-vpn/middleware"
	"just-vpn/model"
	"just-vpn/pkg/redis"
	"just-vpn/pkg/util"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	tempNodeKeyPrefix     = "TEMP_NODE_"
	defaultNodeLinkAESKey = "g9rsnoih20vuiqpw"
)

type NodeRequest struct {
	Code string `json:"code" binding:"required" example:"HK"`
}

type LineAreaResponse struct {
	Country     string `json:"country" example:"香港"`
	Code        string `json:"code" example:"HK"`
	MinConnTime int    `json:"min_conn_time" example:"100"`
	MaxConnTime int    `json:"max_conn_time" example:"300"`
	ImgUrl      string `json:"img_url" example:"https://example.com/hk.png"`
}

type NodeResponse struct {
	LinkUrl string `json:"link_url" example:"x/k5A0v9kiJjL0r3m6X9dA=="` // AES加密后的节点连接数据，解密后需要先base64解码再使用
}

type VpnFlowRequest struct {
	Flow *int `json:"flow" binding:"required" example:"1024"`
}

type HeartbeatResponse struct {
	ConnectStatus int `json:"connect_status" example:"1"` // 连接状态(1=继续连接,0=断开连接)
}

// LinesListHandler 获取国家线路列表
// @Summary 获取国家线路列表
// @Description 获取可用的 VPN 国家或区域线路入口
// @Tags 线路
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response{result=[]LineAreaResponse}
// @Router /lines_list [post]
func LinesListHandler(c *gin.Context) {
	areas, err := model.GetAvailableNodeAreas()
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	result := make([]LineAreaResponse, 0, len(areas))
	for _, area := range areas {
		result = append(result, LineAreaResponse{
			Country:     area.Name,
			Code:        area.Code,
			MinConnTime: area.MinConnTime,
			MaxConnTime: area.MaxConnTime,
			ImgUrl:      area.ImgUrl,
		})
	}

	JsonReturn(c, CodeSuccess, "success", result)
}

// NodeHandler 获取连接节点信息
// @Summary 获取连接节点信息
// @Description 根据国家或区域代码获取本次 VPN 连接节点
// @Tags 线路
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body NodeRequest true "获取连接节点请求"
// @Success 200 {object} Response{result=NodeResponse}
// @Router /node [post]
func NodeHandler(c *gin.Context) {
	var req NodeRequest
	if err := BindMappedJSON(c, &req); err != nil {
		if strings.Contains(err.Error(), "NodeRequest.Code") || strings.Contains(err.Error(), "'Code'") {
			JsonReturn(c, CodeError, "code is required", nil)
			return
		}
		JsonReturn(c, CodeError, "invalid json body", nil)
		return
	}

	code := strings.ToUpper(strings.TrimSpace(req.Code))
	if code == "" {
		JsonReturn(c, CodeError, "code is required", nil)
		return
	}

	node, ok := getNodeForCode(c, code)
	if !ok {
		return
	}

	linkUrl, err := encryptedNodeLinkURL(node.LinkUrl)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	JsonReturn(c, CodeSuccess, "success", NodeResponse{
		LinkUrl: linkUrl,
	})
}

// getNodeForCode 选择节点并保存临时节点，供连接确认和 JSON 配置接口共同使用
func getNodeForCode(c *gin.Context, code string) (model.Node, bool) {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return model.Node{}, false
	}
	if shouldRejectNodeForVipExpired(code, user.VipTime) {
		JsonReturn(c, CodeVipExpired, "vip time expired", nil)
		return model.Node{}, false
	}

	node, reviewNodeEnabled := configuredReviewNode(middleware.CurrentClientInfo(c).Version, code)
	if !reviewNodeEnabled {
		node, err = model.GetAvailableNode(code)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				JsonReturn(c, CodeError, "line is preparing", nil)
				return model.Node{}, false
			}
			JsonReturn(c, CodeError, err.Error(), nil)
			return model.Node{}, false
		}
	}
	if user.IsReal != 1 {
		if err := model.UpdateUserFieldsByID(user.Id, map[string]interface{}{"is_real": 1}); err != nil {
			JsonReturn(c, CodeError, err.Error(), nil)
			return model.Node{}, false
		}
	}
	if err := redis.Set(fmt.Sprintf("%s%d", tempNodeKeyPrefix, user.Id), node, time.Hour); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return model.Node{}, false
	}
	return node, true
}

func encryptedNodeLinkURL(linkUrl string) (string, error) {
	key := strings.TrimSpace(model.ConfigValue(model.ConfigNodeLinkAESKey, defaultNodeLinkAESKey))
	return encryptedNodeLinkURLWithKey(linkUrl, key)
}

func encryptedNodeLinkURLWithKey(linkUrl string, key string) (string, error) {
	switch len(key) {
	case 16, 24, 32:
		encodedLinkUrl := base64.StdEncoding.EncodeToString([]byte(linkUrl))
		return util.AesEnCodeSimple([]byte(encodedLinkUrl), []byte(key)), nil
	default:
		return "", fmt.Errorf("node link aes key length must be 16, 24 or 32")
	}
}

func isFreeNodeCode(code string) bool {
	return code == "FREE" || strings.HasPrefix(code, "FREE_")
}

func shouldRejectNodeForVipExpired(code string, vipTime *time.Time) bool {
	return !isFreeNodeCode(code) && !isValidVipTime(vipTime)
}

// ConnectedHandler 确认 VPN 已连接
// @Summary 确认 VPN 已连接
// @Description 客户端成功连接节点后上报，用于记录当前连接和连接历史
// @Tags 线路
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response
// @Router /connected [post]
func ConnectedHandler(c *gin.Context) {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	node, err := getTempNode(user.Id)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	userPayStatus := userNodePayStatus(user)
	skipHistory, err := model.ShouldSkipDuplicateNodeConnectHistoryInsert(user.Id, node.Id)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	if !skipHistory {
		if err := model.CloseOpenNodeConnectHistory(user.Id); err != nil {
			JsonReturn(c, CodeError, err.Error(), nil)
			return
		}
		if err := model.CreateNodeConnectHistory(buildNodeConnectHistory(c, user, node, userPayStatus)); err != nil {
			JsonReturn(c, CodeError, err.Error(), nil)
			return
		}
		if err := model.AddNodeTodayActiveCount(node.Id); err != nil {
			JsonReturn(c, CodeError, err.Error(), nil)
			return
		}
		if err := model.AddNodeAreaTodayActiveCount(node.Code); err != nil {
			JsonReturn(c, CodeError, err.Error(), nil)
			return
		}
	}

	if err := model.SaveNodeConnectedLog(buildNodeConnectLog(c, user, node, userPayStatus)); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	if err := model.SaveNodeOnlineStatus(user.Id, node); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	JsonReturn(c, CodeSuccess, "success", nil)
}

// HeartbeatHandler VPN 心跳
// @Summary VPN 心跳
// @Description 客户端连接中定时上报，用于刷新当前连接在线时间
// @Tags 线路
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response{result=HeartbeatResponse}
// @Router /heartbeat [post]
func HeartbeatHandler(c *gin.Context) {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	if !isValidVipTime(user.VipTime) {
		if err := disconnectHeartbeatUser(user.Id); err != nil {
			JsonReturn(c, CodeError, err.Error(), nil)
			return
		}
		JsonReturn(c, CodeSuccess, "会员已过期，请断开连接", HeartbeatResponse{ConnectStatus: 0})
		return
	}

	ok, err := model.RefreshNodeOnlineStatus(user.Id)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	if !ok {
		JsonReturn(c, CodeSuccess, "当前没有连接记录", HeartbeatResponse{ConnectStatus: 0})
		return
	}

	JsonReturn(c, CodeSuccess, "success", HeartbeatResponse{ConnectStatus: 1})
}

func disconnectHeartbeatUser(userId int) error {
	if _, err := model.DisconnectNodeConnectLog(userId); err != nil {
		return err
	}
	if err := model.CloseOpenNodeConnectHistory(userId); err != nil {
		return err
	}
	return model.DeleteNodeOnlineStatus(userId)
}

// DisconnectHandler VPN 断开连接
// @Summary VPN 断开连接
// @Description 客户端主动断开 VPN 时上报，未上报时由心跳超时兜底
// @Tags 线路
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} Response
// @Router /disconnect [post]
func DisconnectHandler(c *gin.Context) {
	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	if _, err := model.DisconnectNodeConnectLog(user.Id); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	if err := model.CloseOpenNodeConnectHistory(user.Id); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	if err := model.DeleteNodeOnlineStatus(user.Id); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	if err := redis.Del(tempNodeKey(user.Id)); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	JsonReturn(c, CodeSuccess, "success", nil)
}

// VpnFlowHandler VPN 使用流量上报
// @Summary VPN 使用流量上报
// @Description 客户端上报 VPN 使用流量，单位为 K，用于累加用户总流量和今日流量
// @Tags 线路
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body VpnFlowRequest true "VPN 使用流量上报请求"
// @Success 200 {object} Response
// @Router /vpn_flow [post]
func VpnFlowHandler(c *gin.Context) {
	var req VpnFlowRequest
	if err := BindMappedJSON(c, &req); err != nil {
		if strings.Contains(err.Error(), "VpnFlowRequest.Flow") || strings.Contains(err.Error(), "'Flow'") {
			JsonReturn(c, CodeError, "flow is required", nil)
			return
		}
		JsonReturn(c, CodeError, "invalid json body", nil)
		return
	}
	if req.Flow == nil || *req.Flow <= 0 {
		JsonReturn(c, CodeError, "flow must be greater than 0", nil)
		return
	}

	user, err := middleware.CurrentUser(c)
	if err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	if err := model.AddUserVpnFlow(user.Id, *req.Flow); err != nil {
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}

	JsonReturn(c, CodeSuccess, "success", nil)
}

func getTempNode(userId int) (model.Node, error) {
	var node model.Node
	ok, err := redis.Get(tempNodeKey(userId), &node)
	if err != nil {
		return model.Node{}, err
	}
	if !ok {
		return model.Node{}, fmt.Errorf("please get node first")
	}
	return node, nil
}

func tempNodeKey(userId int) string {
	return fmt.Sprintf("%s%d", tempNodeKeyPrefix, userId)
}

func userNodePayStatus(user model.User) int {
	if isValidVipTime(user.VipTime) {
		return 2
	}
	return 1
}

func buildNodeConnectLog(c *gin.Context, user model.User, node model.Node, userPayStatus int) model.NodeConnectLog {
	return model.NodeConnectLog{
		NodeId:        node.Id,
		Address:       node.Address,
		Ip:            c.ClientIP(),
		Code:          node.Code,
		CountryName:   node.CodeName,
		UserId:        user.Id,
		Platform:      user.Platform,
		UserPayStatus: userPayStatus,
		Status:        1,
	}
}

func buildNodeConnectHistory(c *gin.Context, user model.User, node model.Node, userPayStatus int) model.NodeConnectHistory {
	return model.NodeConnectHistory{
		UserId:        user.Id,
		NodeId:        node.Id,
		Address:       node.Address,
		Ip:            c.ClientIP(),
		Code:          node.Code,
		CountryName:   node.CodeName,
		Platform:      user.Platform,
		UserPayStatus: userPayStatus,
		ConnectedAt:   chinaNow(),
	}
}
