package model

import (
	"testing"
	"time"

	"just-vpn/pkg/redis"
)

func TestPopupDailyShowKeyUsesLocalCalendarDay(t *testing.T) {
	originalLocal := time.Local
	time.Local = time.FixedZone("UTC+8", 8*60*60)
	t.Cleanup(func() { time.Local = originalLocal })

	now := time.Date(2026, time.July, 29, 16, 30, 0, 0, time.UTC)
	if got, want := popupDailyShowKey(12, now), "popup:daily_show:20260730:12"; got != want {
		t.Fatalf("popupDailyShowKey() = %q, want %q", got, want)
	}
}

func TestPopupDailyShowTTLExpiresAfterNextMidnight(t *testing.T) {
	originalLocal := time.Local
	time.Local = time.FixedZone("UTC+8", 8*60*60)
	t.Cleanup(func() { time.Local = originalLocal })

	now := time.Date(2026, time.July, 29, 23, 30, 0, 0, time.Local)
	if got, want := popupDailyShowTTL(now), 90*time.Minute; got != want {
		t.Fatalf("popupDailyShowTTL() = %v, want %v", got, want)
	}
}

func TestUnlimitedPopupDoesNotRequireRedis(t *testing.T) {
	originalRedis := redis.Redis
	redis.Redis = nil
	t.Cleanup(func() { redis.Redis = originalRedis })

	allowed, err := AcquirePopupDailyShow(Popup{MaxDailyShowTimes: 0}, time.Now())
	if err != nil || !allowed {
		t.Fatalf("AcquirePopupDailyShow() = (%v, %v), want (true, nil)", allowed, err)
	}
	if err := ReleasePopupDailyShow(Popup{MaxDailyShowTimes: 0}, time.Now()); err != nil {
		t.Fatalf("ReleasePopupDailyShow() error = %v, want nil", err)
	}
}

func TestLimitedPopupRequiresRedis(t *testing.T) {
	originalRedis := redis.Redis
	redis.Redis = nil
	t.Cleanup(func() { redis.Redis = originalRedis })

	allowed, err := AcquirePopupDailyShow(Popup{MaxDailyShowTimes: 1}, time.Now())
	if err == nil || allowed {
		t.Fatalf("AcquirePopupDailyShow() = (%v, %v), want (false, error)", allowed, err)
	}
}
