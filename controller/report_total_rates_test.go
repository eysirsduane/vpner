package controller

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"just-vpn/model"
	"just-vpn/pkg/redis"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// Catch missing/omitted zero-valued rates in the public JSON contract.
func TestInternalReportTotalRatesIncludeNumericZero(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	internalReportJSON(c, http.StatusOK, CodeSuccess, "success", InternalReportStatsResult{})
	var response struct {
		Result map[string]interface{} `json:"result"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"total_connect_success_rate", "total_payment_success_rate"} {
		if value, ok := response.Result[field].(float64); !ok || value != 0 {
			t.Fatalf("%s must be a numeric zero, got %#v", field, response.Result[field])
		}
	}
}

// Catch using new-user cohorts/order counts for total rates, changing legacy
// fields, failing to cap connection dates, or dividing by a zero denominator.
func TestInternalReportTotalRatesPreserveLegacyFields(t *testing.T) {
	for _, tc := range []struct {
		name            string
		endHours        int
		activeUsers     int64
		connectedUsers  int64
		orderUsers      int64
		paidUsers       int64
		wantConnectRate float64
		wantPaymentRate float64
	}{
		{"partial_day", 12, 20, 15, 8, 3, 75, 37.5},
		{"cross_day_caps_connections_only", 30, 20, 18, 8, 3, 90, 37.5},
		{"empty_denominators", 12, 0, 0, 0, 0, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = db.Close() })
			orm, err := gorm.Open(mysql.New(mysql.Config{Conn: db, SkipInitializeWithVersion: true}), &gorm.Config{})
			if err != nil {
				t.Fatal(err)
			}
			oldDB, oldRedis := model.DB, redis.Redis
			model.DB, redis.Redis = orm, nil
			t.Cleanup(func() { model.DB, redis.Redis = oldDB, oldRedis })
			start := time.Date(2026, 11, 12, 0, 0, 0, 0, time.Local)
			end := start.Add(time.Duration(tc.endHours) * time.Hour)
			mock.ExpectQuery("SELECT \\* FROM `daily_stat`").WithArgs("2026-11-12", 1).
				WillReturnRows(sqlmock.NewRows([]string{"id", "daily_active_users"}).AddRow(1, tc.activeUsers))
			mock.ExpectQuery("(?s)SELECT COUNT\\(\\*\\) FROM `user` WHERE").
				WithArgs(start, end, start, end, start, end).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(10))
			mock.ExpectQuery("(?s)SELECT COUNT\\(DISTINCT u.id\\) AS connect_base_users, COUNT\\(DISTINCT h.user_id\\) AS connected_users").
				WithArgs(start, end, start, end, start, end, start, end).
				WillReturnRows(sqlmock.NewRows([]string{"connect_base_users", "connected_users"}).AddRow(10, 3))
			if tc.activeUsers > 0 {
				connectEnd := end
				if nextDay := start.AddDate(0, 0, 1); connectEnd.After(nextDay) {
					connectEnd = nextDay
				}
				mock.ExpectQuery("SELECT COUNT\\(DISTINCT user_id\\) FROM node_connect_history WHERE connected_at >= \\? AND connected_at < \\?$").
					WithArgs(start, connectEnd).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(tc.connectedUsers))
			}
			mock.ExpectQuery("(?s)SELECT COUNT\\(DISTINCT CASE WHEN u.id IS NOT NULL THEN o.uid END\\) AS created_orders, COUNT\\(DISTINCT CASE WHEN u.id IS NOT NULL AND o.pay_status = \\? THEN o.uid END\\) AS paid_orders, COUNT\\(DISTINCT o.uid\\) AS total_order_users, COUNT\\(DISTINCT CASE WHEN o.pay_status = \\? THEN o.uid END\\) AS total_paid_users FROM `order` o LEFT JOIN `user` u ON u.id = o.uid AND u.create_time >= \\? AND u.create_time < \\? WHERE o.create_time >= \\? AND o.create_time < \\?$").
				WithArgs(model.OrderPayStatusPaid, model.OrderPayStatusPaid, start, end, start, end).
				WillReturnRows(sqlmock.NewRows([]string{"created_orders", "paid_orders", "total_order_users", "total_paid_users"}).AddRow(4, 1, tc.orderUsers, tc.paidUsers))
			mock.ExpectQuery("(?s)SELECT COALESCE\\(SUM\\(classified.money\\)").WithArgs(start, end).
				WillReturnRows(sqlmock.NewRows([]string{"total_income", "apple_income", "third_party_income", "new_purchase_income", "renewal_income"}).
					AddRow(29800, 19900, 9900, 19900, 9900))
			if tc.endHours <= 24 {
				mock.ExpectQuery("SELECT \\* FROM `renewal_tracking`").WithArgs(1, 1).
					WillReturnRows(sqlmock.NewRows([]string{"id", "started_at"}).AddRow(1, start.AddDate(0, 0, 1)))
			}
			result, err := queryInternalReportStats(start, end)
			if err != nil {
				t.Fatal(err)
			}
			data, err := json.Marshal(result)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]interface{}
			if err := json.Unmarshal(data, &fields); err != nil {
				t.Fatal(err)
			}
			if fields["total_connect_success_rate"] != tc.wantConnectRate || fields["total_payment_success_rate"] != tc.wantPaymentRate {
				t.Fatalf("unexpected total rates: %s", data)
			}
			delete(fields, "total_connect_success_rate")
			delete(fields, "total_payment_success_rate")
			wantLegacy := map[string]interface{}{
				"new_users": float64(10), "daily_active_users": float64(tc.activeUsers),
				"connect_base_users": float64(10), "connected_users": float64(3),
				"created_orders": float64(4), "paid_orders": float64(1),
				"new_purchase_income": float64(199), "renewal_income": float64(99),
				"total_income": float64(298), "apple_income": float64(199), "third_party_income": float64(99),
				"due_today_users": float64(0), "renewed_today_due_today_users": float64(0),
				"due_seven_day_users": float64(0), "renewed_today_due_seven_day_users": float64(0),
				"daily_renewal_stats_available": false, "seven_day_renewal_stats_available": false,
			}
			if !reflect.DeepEqual(fields, wantLegacy) {
				t.Fatalf("legacy report fields changed: got %#v, want %#v", fields, wantLegacy)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
