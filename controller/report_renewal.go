package controller

import (
	"log"
	"time"

	"just-vpn/model"
)

func applyInternalRenewalStats(start, end time.Time, result *InternalReportStatsResult) {
	stats, err := model.QueryRenewalReportStats(start, end)
	if err != nil {
		// Optional analytics must not suppress existing income/connection reports.
		log.Printf("renewal report statistics unavailable: %v", err)
		return
	}
	result.RenewalReportStats = stats
}
