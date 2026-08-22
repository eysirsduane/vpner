package controller

import (
	"errors"
	"fmt"

	"just-vpn/middleware"
	"just-vpn/model"
	"just-vpn/pkg/mapping"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// VersionResponse 版本检测响应
type VersionResponse struct {
	HasUpdate bool   `json:"has_update" example:"true"`                       // 是否有新版本
	Name      string `json:"name,omitempty" example:"iOS 1.0.2"`              // 版本名称
	Platform  string `json:"platform,omitempty" example:"iphone"`             // 设备平台(iphone=苹果,android=安卓)
	Version   string `json:"version,omitempty" example:"1.0.2"`               // 展示版本号
	Build     int    `json:"build,omitempty" example:"102"`                   // 数字版本号
	Force     *bool  `json:"force,omitempty" example:"false"`                 // 是否强制更新
	Url       string `json:"url,omitempty" example:"https://example.com/app"` // 下载地址
	Size      string `json:"size,omitempty" example:"23.5MB"`                 // 安装包大小
	Content   string `json:"content,omitempty" example:"修复已知问题"`              // 更新内容
}

// VersionAPIResponse 版本检测接口响应
type VersionAPIResponse struct {
	Code   int             `json:"code" example:"200"`    // 响应状态码
	Msg    string          `json:"msg" example:"success"` // 响应消息
	Result VersionResponse `json:"result"`                // 版本检测结果
}

// VersionHandler 版本检测
// @Summary 版本检测
// @Description 根据请求头中的平台和数字版本号检测是否存在新版本
// @Tags 系统
// @Produce json
// @Security BearerAuth
// @Param X-Platform header string true "设备平台，支持 iphone/android/ios/ipad"
// @Param X-Version header string true "展示版本号"
// @Param X-Build header int true "数字版本号，用于比较"
// @Success 200 {object} VersionAPIResponse
// @Router /version [post]
func VersionHandler(c *gin.Context) {
	client := middleware.CurrentClientInfo(c)
	if client.Platform == "" {
		JsonReturn(c, CodeError, requiredHeaderMessage("platform"), nil)
		return
	}
	if client.Version == "" {
		JsonReturn(c, CodeError, requiredHeaderMessage("version"), nil)
		return
	}
	if client.Build <= 0 {
		JsonReturn(c, CodeError, requiredHeaderMessage("build"), nil)
		return
	}

	version, err := model.GetLatestEnabledVersion(client.Platform)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			JsonReturn(c, CodeSuccess, "success", VersionResponse{HasUpdate: false})
			return
		}
		JsonReturn(c, CodeError, err.Error(), nil)
		return
	}
	if version.Build <= client.Build {
		JsonReturn(c, CodeSuccess, "success", VersionResponse{HasUpdate: false})
		return
	}

	force := version.Force
	JsonReturn(c, CodeSuccess, "success", VersionResponse{
		HasUpdate: true,
		Name:      localizedTextValue(version.Name, client.Language),
		Platform:  version.Platform,
		Version:   version.Version,
		Build:     version.Build,
		Force:     &force,
		Url:       version.Url,
		Size:      version.Size,
		Content:   localizedTextValue(version.Content, client.Language),
	})
}

func requiredHeaderMessage(field string) string {
	return fmt.Sprintf("%s is required", mapping.HeaderField(field))
}
