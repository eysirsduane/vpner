package util

import (
	"bytes"
	"math/rand"
	"time"
)

// RandStr
// 生成不通类型的随机数
func RandStr(k string, n int) string {
	length := 0
	pattern := ""
	switch k {
	case "n":
		pattern = "1234567890"
		length = 10
		break
	case "s":
		pattern = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLOMNOPQRSTUVWXYZ"
		length = 52
		break
	case "r":
		pattern = "1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLOMNOPQRSTUVWXYZ"
		length = 62
		break
	case "y":
		pattern = "1234567890abcdefghijklmnopqrstuvwxyz"
		length = 35
		break
	case "a":
		pattern = "1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLOMNOPQRSTUVWXYZ"
		length = 62
		break
	default:
		pattern = "1234567890"
		length = 10
		break
	}
	str := ""
	rand.NewSource(time.Now().UnixNano()) // 产生随机种子
	var s bytes.Buffer
	for i := 0; i < n; i++ {
		s.WriteByte(pattern[rand.Int63()%int64(length)])
	}
	str = s.String()
	return str
}
