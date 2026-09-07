package model

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	appredis "just-vpn/pkg/redis"
	"just-vpn/pkg/setting"
)

var dailyH5AppleLimitMu sync.Mutex

const (
	dailyH5AppleLimitKeyPrefix = "PAY_H5_APPLE_DAILY_LIMIT"
	dailyH5AppleLimitTTL       = 96 * time.Hour
)

// DailyH5AppleLimitResult 记录某天内购限额的计算依据，金额单位均为分。
type DailyH5AppleLimitResult struct {
	TargetDate    time.Time
	RevenueStart  time.Time
	RevenueEnd    time.Time
	TotalRevenue  int64
	MinimumAmount int64
	Percent       int
	LimitAmount   int64
}

// TodayH5AppleLimit 返回当前北京时间日期的内购限额。
// Redis 缺失时自动使用完整的前一日收入补算并写回，确保服务重启后规则仍可用。
func TodayH5AppleLimit(now time.Time) (int64, error) {
	now = now.In(time.Local)
	minimumAmount := h5AppleMinimumAmount()
	percent := h5AppleAmountPercent()
	if minimumAmount == 0 && percent == 0 {
		return 0, nil
	}

	targetDate := startOfLocalDay(now)
	key := dailyH5AppleLimitKey(targetDate, minimumAmount, percent)
	var limit int64
	found, err := appredis.Get(key, &limit)
	if err != nil {
		return 0, err
	}
	if found {
		if limit < 0 {
			return 0, fmt.Errorf("daily h5 apple limit is invalid")
		}
		return limit, nil
	}

	dailyH5AppleLimitMu.Lock()
	defer dailyH5AppleLimitMu.Unlock()
	minimumAmount = h5AppleMinimumAmount()
	percent = h5AppleAmountPercent()
	if minimumAmount == 0 && percent == 0 {
		return 0, nil
	}
	key = dailyH5AppleLimitKey(targetDate, minimumAmount, percent)
	if found, err = appredis.Get(key, &limit); err != nil {
		return 0, err
	} else if found {
		if limit < 0 {
			return 0, fmt.Errorf("daily h5 apple limit is invalid")
		}
		return limit, nil
	}

	revenueStart := targetDate.AddDate(0, 0, -1)
	result, err := calculateAndStoreDailyH5AppleLimit(targetDate, revenueStart, targetDate, minimumAmount, percent)
	if err != nil {
		return 0, err
	}
	return result.LimitAmount, nil
}

// CalculateAndStoreDailyH5AppleLimit 按指定收入区间计算目标日期的限额并写入 Redis。
func CalculateAndStoreDailyH5AppleLimit(targetDate, revenueStart, revenueEnd time.Time) (DailyH5AppleLimitResult, error) {
	dailyH5AppleLimitMu.Lock()
	defer dailyH5AppleLimitMu.Unlock()
	return calculateAndStoreDailyH5AppleLimit(
		targetDate,
		revenueStart,
		revenueEnd,
		h5AppleMinimumAmount(),
		h5AppleAmountPercent(),
	)
}

func calculateAndStoreDailyH5AppleLimit(targetDate, revenueStart, revenueEnd time.Time, minimumAmount int64, percent int) (DailyH5AppleLimitResult, error) {
	targetDate = startOfLocalDay(targetDate)
	revenueStart = revenueStart.In(time.Local)
	revenueEnd = revenueEnd.In(time.Local)
	if !revenueEnd.After(revenueStart) {
		return DailyH5AppleLimitResult{}, fmt.Errorf("daily h5 apple revenue range is invalid")
	}

	var totalRevenue int64
	if percent > 0 {
		var err error
		totalRevenue, err = TotalPaidAmountBetween(revenueStart, revenueEnd)
		if err != nil {
			return DailyH5AppleLimitResult{}, err
		}
	}
	limit := ComputeDailyH5AppleLimit(totalRevenue, minimumAmount, percent)
	if err := appredis.Set(dailyH5AppleLimitKey(targetDate, minimumAmount, percent), limit, dailyH5AppleLimitTTL); err != nil {
		return DailyH5AppleLimitResult{}, err
	}
	return DailyH5AppleLimitResult{
		TargetDate:    targetDate,
		RevenueStart:  revenueStart,
		RevenueEnd:    revenueEnd,
		TotalRevenue:  totalRevenue,
		MinimumAmount: minimumAmount,
		Percent:       percent,
		LimitAmount:   limit,
	}, nil
}

// TotalPaidAmountBetween 汇总时间区间内全部已支付订单收入，不区分苹果或三方渠道。
func TotalPaidAmountBetween(start, end time.Time) (int64, error) {
	var total int64
	err := DB.Model(&Order{}).
		Select("COALESCE(SUM(money), 0)").
		Where("pay_status = ?", OrderPayStatusPaid).
		Where("pay_time >= ? AND pay_time < ?", start, end).
		Scan(&total).Error
	return total, err
}

func ComputeDailyH5AppleLimit(totalRevenue, minimumAmount int64, percent int) int64 {
	if totalRevenue < 0 {
		totalRevenue = 0
	}
	if minimumAmount < 0 {
		minimumAmount = 0
	}
	if percent < 0 || percent > 100 {
		percent = 0
	}
	dynamicAmount := totalRevenue/100*int64(percent) + totalRevenue%100*int64(percent)/100
	if dynamicAmount < minimumAmount {
		return minimumAmount
	}
	return dynamicAmount
}

func h5AppleMinimumAmount() int64 {
	value := strings.TrimSpace(PayConfigValue(PayConfigH5AppleAmount, "0"))
	amount, err := strconv.ParseInt(value, 10, 64)
	if err != nil || amount < 0 {
		return 0
	}
	return amount
}

func h5AppleAmountPercent() int {
	value := strings.TrimSpace(PayConfigValue(PayConfigH5AppleAmountPercent, "0"))
	return parseH5AppleAmountPercent(value)
}

func parseH5AppleAmountPercent(value string) int {
	percent, err := strconv.Atoi(value)
	if err != nil || percent <= 0 || percent > 100 {
		return 0
	}
	return percent
}

func dailyH5AppleLimitKey(targetDate time.Time, minimumAmount int64, percent int) string {
	product := strings.TrimSpace(setting.AppConfig.Product)
	return fmt.Sprintf(
		"%s:%s:%s:%d:%d",
		dailyH5AppleLimitKeyPrefix,
		product,
		targetDate.In(time.Local).Format("20060102"),
		minimumAmount,
		percent,
	)
}

func startOfLocalDay(value time.Time) time.Time {
	value = value.In(time.Local)
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.Local)
}
