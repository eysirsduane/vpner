package model

import (
	"context"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Internal payment snapshots. Never expose these fields through client DTOs.
type RenewalOrderSnapshot struct {
	RenewalRecorded bool       `json:"-" gorm:"column:renewal_recorded;not null;default:false"`
	RenewalWasPaid  bool       `json:"-" gorm:"column:renewal_was_paid;not null;default:false"`
	VipBefore       *time.Time `json:"-" gorm:"column:vip_before;type:datetime;index:idx_order_vip_before"`
	VipAfter        *time.Time `json:"-" gorm:"column:vip_after;type:datetime"`
}

type RenewalTracking struct {
	ID        int       `gorm:"primaryKey;autoIncrement:false"`
	StartedAt time.Time `gorm:"type:datetime;not null"`
}

func (RenewalTracking) TableName() string { return "renewal_tracking" }

func InitRenewalTracking() error {
	return DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&RenewalTracking{ID: 1, StartedAt: time.Now()}).Error
}

func HasPaidOrder(tx *gorm.DB, userID int) (bool, error) {
	var paid bool
	err := tx.Raw("SELECT EXISTS(SELECT 1 FROM `order` WHERE uid = ? AND pay_status = ?)", userID, OrderPayStatusPaid).Scan(&paid).Error
	return paid, err
}

type RenewalReportStats struct {
	DueTodayUsers             int64 `json:"due_today_users"`
	RenewedTodayDueTodayUsers int64 `json:"renewed_today_due_today_users"`
	DueSevenDayUsers          int64 `json:"due_seven_day_users"`
	// Legacy wire names retained: counts use expiry cohorts, not payment dates.
	RenewedTodayDueSevenDayUsers  int64 `json:"renewed_today_due_seven_day_users"`
	DailyRenewalStatsAvailable    bool  `json:"daily_renewal_stats_available"`
	SevenDayRenewalStatsAvailable bool  `json:"seven_day_renewal_stats_available"`
}

// Payment snapshots preserve original expiry cohorts, including advance renewals.
// Both rates select users by original expiry date. Any successful renewal before
// the report cutoff qualifies, even when paid before the expiry window begins.
// Both sides count distinct users; unrelated subsequent purchases do not add users.
const renewalCohortSQL = `SELECT
 COUNT(DISTINCT CASE WHEN c.expires_at >= ? THEN c.uid END) AS due_today_users,
 COUNT(DISTINCT CASE WHEN p.vip_before >= ? THEN c.uid END) AS renewed_today_due_today_users,
 COUNT(DISTINCT c.uid) AS due_seven_day_users,
 COUNT(DISTINCT CASE WHEN p.uid IS NOT NULL THEN c.uid END) AS renewed_today_due_seven_day_users
FROM (
 SELECT id AS uid, vip_time AS expires_at FROM ` + "`user`" + `
 WHERE vip_time >= ? AND vip_time < ?
 AND EXISTS(SELECT 1 FROM ` + "`order`" + ` paid WHERE paid.uid=` + "`user`" + `.id AND paid.pay_status=3)
 UNION
 SELECT uid, vip_before AS expires_at FROM ` + "`order`" + `
 WHERE renewal_recorded=1 AND renewal_was_paid=1 AND pay_status IN (3,4)
 AND vip_before >= ? AND vip_before < ?
) c
LEFT JOIN (
 SELECT DISTINCT uid, vip_before FROM ` + "`order`" + `
 WHERE pay_status=3 AND renewal_recorded=1 AND renewal_was_paid=1
 AND pay_time < ?
 AND vip_before >= ? AND vip_before < ?
) p ON p.uid=c.uid`

func renewalCoverage(start, trackingSince time.Time) (daily, weekly bool) {
	firstDay := time.Date(trackingSince.Year(), trackingSince.Month(), trackingSince.Day(), 0, 0, 0, 0, trackingSince.Location())
	if trackingSince.After(firstDay) {
		firstDay = firstDay.AddDate(0, 0, 1)
	}
	return !start.Before(firstDay), !start.AddDate(0, 0, -6).Before(firstDay)
}

func QueryRenewalReportStats(start, end time.Time) (RenewalReportStats, error) {
	var result RenewalReportStats
	dayStart := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	dayEnd := dayStart.AddDate(0, 0, 1)
	if !start.Equal(dayStart) || !end.After(start) || end.After(dayEnd) {
		return result, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	db := DB.WithContext(ctx)
	var tracking RenewalTracking
	if err := db.First(&tracking, 1).Error; err != nil {
		return result, err
	}
	result.DailyRenewalStatsAvailable, result.SevenDayRenewalStatsAvailable = renewalCoverage(start, tracking.StartedAt.In(start.Location()))
	if !result.DailyRenewalStatsAvailable {
		return result, nil
	}
	weekStart := start.AddDate(0, 0, -6)
	// A rolling deployment may still have old writers. Never report fabricated
	// precision when paid orders in a supposedly covered window lack snapshots.
	var gaps struct {
		Daily  int64
		Weekly int64
	}
	if err := db.Raw(`SELECT COUNT(CASE WHEN pay_time >= ? THEN 1 END) AS daily,COUNT(*) AS weekly FROM `+"`order`"+` WHERE pay_status IN (3,4) AND renewal_recorded=0 AND pay_time >= ? AND pay_time < ?`, start, weekStart, end).Scan(&gaps).Error; err != nil {
		return RenewalReportStats{}, err
	}
	result.DailyRenewalStatsAvailable = result.DailyRenewalStatsAvailable && gaps.Daily == 0
	result.SevenDayRenewalStatsAvailable = result.SevenDayRenewalStatsAvailable && gaps.Weekly == 0
	var counts RenewalReportStats
	if err := db.Raw(renewalCohortSQL, start, start, weekStart, dayEnd, weekStart, dayEnd, end, weekStart, dayEnd).Scan(&counts).Error; err != nil {
		return RenewalReportStats{}, err
	}
	counts.DailyRenewalStatsAvailable = result.DailyRenewalStatsAvailable
	counts.SevenDayRenewalStatsAvailable = result.SevenDayRenewalStatsAvailable
	return counts, nil
}
