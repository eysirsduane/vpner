package model

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestRenewalQueryCoverageAndAggregateMapping(t *testing.T) {
	start := time.Date(2026, 9, 13, 0, 0, 0, 0, time.FixedZone("CN", 8*3600))
	end := start.Add(12 * time.Hour)
	for _, tc := range []struct {
		name                string
		tracking            time.Time
		dailyGap, weeklyGap int64
		daily, weekly       bool
		query               bool
	}{
		{"ready", start.AddDate(0, 0, -20), 0, 0, true, true, true},
		{"old writer today", start.AddDate(0, 0, -20), 1, 1, false, false, true},
		{"old writer last week", start.AddDate(0, 0, -20), 0, 1, true, false, true},
		{"new tracking", start.Add(time.Hour), 0, 0, false, false, false},
		{"daily ready weekly warming", start.AddDate(0, 0, -2), 0, 0, true, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			orm, err := gorm.Open(mysql.New(mysql.Config{Conn: db, SkipInitializeWithVersion: true}), &gorm.Config{})
			if err != nil {
				t.Fatal(err)
			}
			old := DB
			DB = orm
			defer func() { DB = old }()
			mock.ExpectQuery("SELECT \\* FROM `renewal_tracking`").WillReturnRows(sqlmock.NewRows([]string{"id", "started_at"}).AddRow(1, tc.tracking))
			if tc.query {
				mock.ExpectQuery("SELECT COUNT\\(CASE").WithArgs(start, start.AddDate(0, 0, -6), end).WillReturnRows(sqlmock.NewRows([]string{"daily", "weekly"}).AddRow(tc.dailyGap, tc.weeklyGap))
				mock.ExpectQuery("SELECT COUNT\\(DISTINCT").WithArgs(start, start, start.AddDate(0, 0, -6), start.AddDate(0, 0, 1), start.AddDate(0, 0, -6), start.AddDate(0, 0, 1), end, start.AddDate(0, 0, -6), start.AddDate(0, 0, 1)).WillReturnRows(sqlmock.NewRows([]string{"due_today_users", "renewed_today_due_today_users", "due_seven_day_users", "renewed_today_due_seven_day_users"}).AddRow(4, 1, 8, 2))
			}
			result, err := QueryRenewalReportStats(start, end)
			if err != nil || result.DailyRenewalStatsAvailable != tc.daily || result.SevenDayRenewalStatsAvailable != tc.weekly {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if tc.query && (result.DueTodayUsers != 4 || result.RenewedTodayDueTodayUsers != 1 || result.DueSevenDayUsers != 8 || result.RenewedTodayDueSevenDayUsers != 2) {
				t.Fatalf("counts lost: %+v", result)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
