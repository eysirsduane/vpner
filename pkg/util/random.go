package util

import (
	"math/rand"
	"strings"
	"time"
)

// RandomString
// 取指定长度的随机字符串
func RandomString(length int) string {
	if length < 1 {
		return ""
	}
	char := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	charArr := strings.Split(char, "")
	charLen := len(charArr)
	ran := rand.New(rand.NewSource(time.Now().UnixNano()))
	rChar := ""
	for i := 1; i <= length; i++ {
		rChar = rChar + charArr[ran.Intn(charLen)]
	}
	return rChar
}

// RandomNumber
// 取指定长度的随机字符串
func RandomNumber(length int) string {
	if length < 1 {
		return ""
	}
	char := "0123456789"
	charArr := strings.Split(char, "")
	charLen := len(charArr)
	ran := rand.New(rand.NewSource(time.Now().UnixNano()))
	rChar := ""
	for i := 1; i <= length; i++ {
		rChar = rChar + charArr[ran.Intn(charLen)]
	}
	return rChar
}

// RandomInt
// 随机取范围内数值 [start,end]
func RandomInt(start int, end int) int {
	rand.Seed(time.Now().UnixNano())
	random := rand.Intn(1 + end - start)
	random = start + random
	return random
}

// RandomSeedInt
// 随机取范围内数值 [start,end]
func RandomSeedInt(start int, end int, randSeed int64) int {
	rand.Seed(randSeed)
	random := rand.Intn(1 + end - start)
	random = start + random
	return random
}

// RandomMobileCode
// 随机手机验证码
func RandomMobileCode(length int) string {
	if length < 1 {
		return ""
	}
	char := "0123456789"
	charArr := strings.Split(char, "")
	charLen := len(charArr)
	ran := rand.New(rand.NewSource(time.Now().UnixNano()))
	rChar := ""
	for i := 1; i <= length; i++ {
		rChar = rChar + charArr[ran.Intn(charLen)]
	}
	return rChar
}
