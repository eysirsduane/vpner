package controller

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"just-vpn/model"
)

func TestRenewalStatisticsFailurePreservesExistingReport(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	orm, err := gorm.Open(mysql.New(mysql.Config{Conn: db, SkipInitializeWithVersion: true}), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	old := model.DB
	model.DB = orm
	defer func() { model.DB = old }()
	mock.ExpectQuery("SELECT \\* FROM `renewal_tracking`").WillReturnError(errors.New("fixture unavailable"))
	start := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	result := InternalReportStatsResult{NewUsers: 12, DailyActiveUsers: 100, TotalIncome: 300, AppleIncome: 200, ThirdPartyIncome: 100}
	applyInternalRenewalStats(start, start.Add(time.Hour), &result)
	if result.NewUsers != 12 || result.TotalIncome != 300 || result.AppleIncome != 200 || result.ThirdPartyIncome != 100 || result.DailyActiveUsers != 100 || result.DailyRenewalStatsAvailable || result.SevenDayRenewalStatsAvailable {
		t.Fatalf("original report changed: %+v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}
