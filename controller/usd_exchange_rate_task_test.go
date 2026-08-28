package controller

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
)

func TestParseSafeUSDCNYCentralParity(t *testing.T) {
	page := `
		<tr><td>2026-08-28</td><td>678.11</td><td>786.83</td></tr>
		<tr><td>2026-08-27</td><td>678.40</td><td>787.39</td></tr>`

	rate, err := parseSafeUSDCNYCentralParity(page, "2026-08-28")
	if err != nil {
		t.Fatalf("parseSafeUSDCNYCentralParity returned error: %v", err)
	}
	if rate != 6.7811 {
		t.Fatalf("rate = %v, want 6.7811", rate)
	}
}

func TestParseSafeUSDCNYCentralParityRejectsMissingOrUnsafeRate(t *testing.T) {
	if _, err := parseSafeUSDCNYCentralParity(`<tr><td>2026-08-28</td><td>1200</td></tr>`, "2026-08-28"); err == nil {
		t.Fatal("expected unsafe rate to be rejected")
	}
	if _, err := parseSafeUSDCNYCentralParity(`<tr><td>2026-08-27</td><td>678.11</td></tr>`, "2026-08-28"); err == nil {
		t.Fatal("expected missing date to be rejected")
	}
}

func TestUSDExchangeRateUpdateShouldScheduleAtConfiguredTime(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	original := common.OptionMap
	common.OptionMap = map[string]string{usdExchangeRateAutoUpdateEnabledOption: "true"}
	common.OptionMapRWMutex.Unlock()
	t.Cleanup(func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = original
		common.OptionMapRWMutex.Unlock()
	})

	handler := usdExchangeRateUpdateHandler{}
	loc := time.FixedZone("CST", 8*60*60)
	if handler.ShouldSchedule(time.Date(2026, time.August, 28, 9, 19, 0, 0, loc), nil) {
		t.Fatal("must not schedule before 09:20")
	}
	if !handler.ShouldSchedule(time.Date(2026, time.August, 28, 9, 20, 0, 0, loc), nil) {
		t.Fatal("must schedule at 09:20 on a workday")
	}
	if handler.ShouldSchedule(time.Date(2026, time.August, 29, 9, 20, 0, 0, loc), nil) {
		t.Fatal("must not schedule on Saturday")
	}
	if handler.ShouldSchedule(time.Date(2026, time.August, 28, 18, 0, 0, 0, loc), nil) {
		t.Fatal("must stop retrying after the trading day")
	}
}
