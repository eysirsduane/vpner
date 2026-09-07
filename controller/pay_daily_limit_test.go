package controller

import (
	"errors"
	"testing"
	"time"
)

func TestShouldForceAppleByTodayAmountUsesDailyLimit(t *testing.T) {
	originalLimit := payTodayH5AppleLimit
	originalAmount := payTodayApplePaidAmount
	t.Cleanup(func() {
		payTodayH5AppleLimit = originalLimit
		payTodayApplePaidAmount = originalAmount
	})

	payTodayH5AppleLimit = func(time.Time) (int64, error) { return 3000, nil }
	payTodayApplePaidAmount = func(time.Time) (int, error) { return 2999, nil }
	matched, reason, err := shouldForceAppleByTodayAmount(payContext{now: time.Now()})
	if err != nil || !matched || reason != "today_apple_amount_not_reached:2999/3000" {
		t.Fatalf("below limit: matched=%v reason=%q err=%v", matched, reason, err)
	}

	payTodayApplePaidAmount = func(time.Time) (int, error) { return 3000, nil }
	matched, reason, err = shouldForceAppleByTodayAmount(payContext{now: time.Now()})
	if err != nil || matched || reason != "today_apple_amount_reached:3000/3000" {
		t.Fatalf("reached limit: matched=%v reason=%q err=%v", matched, reason, err)
	}

	payTodayH5AppleLimit = func(time.Time) (int64, error) { return 0, nil }
	matched, reason, err = shouldForceAppleByTodayAmount(payContext{now: time.Now()})
	if err != nil || matched || reason != "h5_threshold_disabled" {
		t.Fatalf("disabled limit: matched=%v reason=%q err=%v", matched, reason, err)
	}

	payTodayH5AppleLimit = func(time.Time) (int64, error) { return 0, errors.New("redis failed") }
	matched, reason, err = shouldForceAppleByTodayAmount(payContext{now: time.Now()})
	if err == nil || !matched || reason != "daily_h5_apple_limit_query_error" {
		t.Fatalf("limit error: matched=%v reason=%q err=%v", matched, reason, err)
	}
}
