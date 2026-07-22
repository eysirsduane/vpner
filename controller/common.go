package controller

import (
	"encoding/json"
	"just-vpn/pkg/errmsg"
	"just-vpn/pkg/mapping"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

const (
	CodeSuccess    = 200
	CodeVipExpired = 901
)

type Response struct {
	Code   int         `json:"code" example:"200"`
	Msg    string      `json:"msg" example:"success"`
	Result interface{} `json:"result"`
}

func JsonReturn(c *gin.Context, code int, msg string, result interface{}) {
	if result == nil {
		result = gin.H{}
	}
	if code != CodeSuccess {
		msg = errmsg.Friendly(msg)
	}
	originalPath := mapping.OriginalEndpoint(c.FullPath())
	response := map[string]interface{}{
		"code":   code,
		"msg":    msg,
		"result": result,
	}
	c.JSON(http.StatusOK, mapping.MapResponse(originalPath, response))
}

func BindMappedJSON(c *gin.Context, dst interface{}) error {
	var body map[string]interface{}
	if err := c.ShouldBindJSON(&body); err != nil {
		return err
	}

	originalPath := mapping.OriginalEndpoint(c.FullPath())
	normalizedBody := normalizeMappedRequestBody(originalPath, body)
	data, err := json.Marshal(normalizedBody)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, dst); err != nil {
		return err
	}
	if binding.Validator == nil {
		return nil
	}
	return binding.Validator.ValidateStruct(dst)
}

func normalizeMappedRequestBody(originalPath string, body map[string]interface{}) map[string]interface{} {
	fields := mapping.RequestFields(originalPath)
	if len(fields) == 0 {
		return body
	}

	reverseFields := make(map[string]string, len(fields))
	for originalField, mappedField := range fields {
		if mappedField == "" {
			continue
		}
		reverseFields[mappedField] = originalField
	}

	normalizedBody := make(map[string]interface{}, len(body))
	for field, value := range body {
		if originalField, ok := reverseFields[field]; ok {
			normalizedBody[originalField] = value
			continue
		}
		normalizedBody[field] = value
	}
	return normalizedBody
}
