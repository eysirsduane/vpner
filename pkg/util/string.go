package util

import (
	"github.com/unknwon/com"
	"strconv"
	"strings"
	"time"
)

// 手机号加*
func PhoneFmt(phone string) string {
	slice := []byte(phone)
	return string(slice[0:3]) + "****" + string(slice[7:])
}

// 去除字符串两端空格
func Trim(str string) string {
	return strings.Trim(str, " ")
}

// 检查字符串切片是否包含某个值
func IsExist(list []string, val string) bool {
	for _, item := range list {
		if item == val {
			return true
		}
	}
	return false
}

// 检查字符串切片是否包含某个值
func IsExistInt(list []int, val int) bool {
	for _, item := range list {
		if item == val {
			return true
		}
	}
	return false
}

// 获取全局唯一order_id
func GetOrderId() string {
	formatLayout := "20060102030405"
	orderNo := time.Now().Format(formatLayout)
	r := RandInt(1000, 9999)
	return orderNo + com.ToStr(r)
}

func Sif(condition bool, trueVal, falseVal string) string {
	if condition {
		return trueVal
	}
	return falseVal
}

func StoF(s string) float64 {
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return v
	}
	return 0.0
}

func ItoS(s int) string {
	v := strconv.Itoa(s)
	return v
}
func StoI(s string) int {
	v, err := strconv.Atoi(s)
	if err == nil {
		return v
	}
	return 0
}

// 转换html标记
func HtmlEs(s string) string {
	s = strings.Replace(s, "&lt;", "<", -1)
	s = strings.Replace(s, "&gt;", ">", -1)
	s = strings.Replace(s, "&quot;", "\"", -1)
	s = strings.Replace(s, "#x27;", "\\", -1)
	s = strings.Replace(s, "&#x60;", "`", -1)
	s = strings.Replace(s, "&amp;", "&", -1)
	return s
}
