package task

import (
	"log"
	"time"

	"just-vpn/model"
)

const dailyH5AppleLimitHour = 23
const dailyH5AppleLimitMinute = 59

// StartPayDailyLimitTask 启动内购每日动态限额任务。
// 启动时确保当天限额存在，每天北京时间 23:59 预计算下一天限额。
func StartPayDailyLimitTask() {
	go func() {
		now := time.Now().In(time.Local)
		ensureTodayH5AppleLimit(now)
		if now.Hour() == dailyH5AppleLimitHour && now.Minute() >= dailyH5AppleLimitMinute {
			calculateTomorrowH5AppleLimit(now)
		}

		for {
			now = time.Now().In(time.Local)
			nextRun := nextDailyH5AppleLimitRun(now)
			time.Sleep(time.Until(nextRun))
			calculateTomorrowH5AppleLimit(nextRun)
		}
	}()
}

func ensureTodayH5AppleLimit(now time.Time) {
	limit, err := model.TodayH5AppleLimit(now)
	if err != nil {
		log.Printf("ensure today h5 apple limit failed: date=%s err=%v", now.Format("2006-01-02"), err)
		return
	}
	log.Printf("today h5 apple limit ready: date=%s limit=%d", now.Format("2006-01-02"), limit)
}

func calculateTomorrowH5AppleLimit(now time.Time) {
	now = now.In(time.Local)
	revenueStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	targetDate := revenueStart.AddDate(0, 0, 1)
	result, err := model.CalculateAndStoreDailyH5AppleLimit(targetDate, revenueStart, now)
	if err != nil {
		log.Printf("calculate tomorrow h5 apple limit failed: target_date=%s err=%v", targetDate.Format("2006-01-02"), err)
		return
	}
	log.Printf(
		"tomorrow h5 apple limit ready: target_date=%s total_revenue=%d percent=%d minimum=%d limit=%d",
		result.TargetDate.Format("2006-01-02"),
		result.TotalRevenue,
		result.Percent,
		result.MinimumAmount,
		result.LimitAmount,
	)
}

func nextDailyH5AppleLimitRun(now time.Time) time.Time {
	now = now.In(time.Local)
	next := time.Date(now.Year(), now.Month(), now.Day(), dailyH5AppleLimitHour, dailyH5AppleLimitMinute, 0, 0, time.Local)
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}
