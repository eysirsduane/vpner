package util

import (
	"strconv"
	"time"
)

//获取当前时间戳
func GetTime(t string) string {
	now_time := time.Now().UnixNano()/1e6 + 120*1000
	if t == "13" {
		return strconv.FormatInt(now_time, 10)
	}
	now_time = time.Now().Unix()
	return strconv.FormatInt(now_time, 10)
}

// 获取当前时间 int
func GetNowInt() int {
	return int(time.Now().Unix())
}

// 获取当前时间
func GetNowTimeStr() string {
	return strconv.Itoa(int(time.Now().Unix()))
}

// 获取当前时间年月日时分秒
func NowTimeYmd() string {
	formatLayout := "20060102030405"
	orderNo := time.Now().Format(formatLayout)
	return orderNo
}

//获取当前时间
func GetTimeStr(t int, formate string) string {
	now_time := time.Now()
	formateStr := "20060102150405"
	if formate == "Y-m-d" {
		formateStr = "2006-01-02"
	}
	if formate == "Ymd" {
		formateStr = "20060102"
	}
	if formate == "Y-m-d H:i" {
		formateStr = "2006-01-02 15:04"
	}
	if formate == "Y-m-d H:i:s" {
		formateStr = "2006-01-02 15:04:05"
	}
	if formate == "Y-m-d H" {
		formateStr = "2006-01-02 15"
	}
	if formate == "Y-m" {
		formateStr = "2006-01"
	}
	if formate == "H" {
		formateStr = "15"
	}
	if t > 0 {
		now_time = time.Unix(int64(t), 0)
	}
	return now_time.Format(formateStr)
}

func GetTodayTime() int {
	the_time, _ := time.ParseInLocation("2006-01-02", GetTimeStr(0, "Y-m-d"), time.Local)
	return int(the_time.Unix())
}

//获取今天0点0时0分的时间戳
func TodayStart() int {
	currentTime := time.Now()
	startTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, currentTime.Location())
	return int(startTime.Unix())
}

//获取今天23:59:59秒的时间戳
func TodayEnd() int {
	currentTime := time.Now()
	endTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 23, 59, 59, 0, currentTime.Location())
	return int(endTime.Unix())
}

// int 秒 转时间量
func GetValTime(seconds int) map[string]interface{} {
	dv := 86400
	dh := 3600
	dm := 60
	day := seconds / dv
	hour := (seconds % dv) / dh
	min := (seconds % dh) / dm
	sec := seconds % dm
	return map[string]interface{}{
		"day":    day,
		"hour":   hour,
		"minute": min,
		"second": sec,
	}
}

// 时间日期转时间戳
func GetTimeStamp(date, formate string) (timeStamp string) {
	if formate == "" {
		formate = "Y-m-d H:i:s"
	}
	formateStr := "20060102150405"
	if formate == "Y-m-d" {
		formateStr = "2006-01-02"
	}
	if formate == "Ymd" {
		formateStr = "20060102"
	}
	if formate == "Y-m-d H:i" {
		formateStr = "2006-01-02 15:04"
	}
	if formate == "Y-m-d H:i:s" {
		formateStr = "2006-01-02 15:04:05"
	}
	the_time, _ := time.ParseInLocation(formateStr, date, time.Local)
	times := the_time.Unix()

	if times > 0 {
		return strconv.FormatInt(times, 10)
	}
	return "0"
}
