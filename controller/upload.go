package controller

import (
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

var uploadRoot = "upload"

// UploadFileHandler 获取上传静态资源
// @Summary 获取上传静态资源
// @Description 从 upload 目录读取图片或其他静态资源
// @Tags 系统
// @Produce octet-stream
// @Param filepath path string true "资源相对路径"
// @Success 200 {file} file
// @Router /upload/{filepath} [get]
func UploadFileHandler(c *gin.Context) {
	filePath := strings.TrimPrefix(c.Param("filepath"), "/")
	filePath = path.Clean("/" + filePath)
	filePath = strings.TrimPrefix(filePath, "/")
	if filePath == "." || filePath == "" {
		c.Status(http.StatusNotFound)
		return
	}

	fs := http.Dir(uploadRoot)
	file, err := fs.Open(filePath)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil || stat.IsDir() {
		c.Status(http.StatusNotFound)
		return
	}

	c.FileFromFS(filePath, fs)
}
