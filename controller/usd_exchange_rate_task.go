package controller

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

const (
	usdExchangeRateAutoUpdateEnabledOption = "USDExchangeRateAutoUpdateEnabled"
	usdExchangeRateLastSourceOption        = "USDExchangeRateLastSource"
	usdExchangeRateLastSourceDateOption    = "USDExchangeRateLastSourceDate"
	usdExchangeRateLastUpdatedAtOption     = "USDExchangeRateLastUpdatedAt"
	usdExchangeRateSource                  = "CFETS/SAFE"
	usdExchangeRateRetryInterval           = 30 * time.Minute
	usdExchangeRateRequestTimeout          = 15 * time.Second
)

var (
	shanghaiLocation, _           = time.LoadLocation("Asia/Shanghai")
	safeUSDExchangeRateHTTPClient = &http.Client{Timeout: usdExchangeRateRequestTimeout}
	fetchSafeUSDExchangeRate      = fetchSafeUSDExchangeRateForDate
)

// usdExchangeRateUpdateHandler updates the display exchange rate from the
// CFETS RMB central parity rate published by SAFE. It intentionally does not
// change Price, which is the operator's recharge selling price.
type usdExchangeRateUpdateHandler struct{}

func (usdExchangeRateUpdateHandler) Type() string {
	return model.SystemTaskTypeUSDExchangeRateUpdate
}

func (usdExchangeRateUpdateHandler) Enabled() bool {
	return systemOptionIsTrue(usdExchangeRateAutoUpdateEnabledOption)
}

func (usdExchangeRateUpdateHandler) Interval() time.Duration {
	return usdExchangeRateRetryInterval
}

func (usdExchangeRateUpdateHandler) NewPayload() any { return nil }

// ShouldSchedule schedules once the 09:15 CFETS publication has had five
// minutes to reach SAFE, then retries every 30 minutes only after failures.
func (usdExchangeRateUpdateHandler) ShouldSchedule(now time.Time, latest *model.SystemTask) bool {
	loc := shanghaiLocation
	if loc == nil {
		loc = time.FixedZone("CST", 8*60*60)
	}
	now = now.In(loc)
	if now.Weekday() == time.Saturday || now.Weekday() == time.Sunday {
		return false
	}
	if now.Hour() < 9 || (now.Hour() == 9 && now.Minute() < 20) {
		return false
	}
	// A missing value after the trading day is most likely a public holiday.
	// Try again on the next workday instead of generating retries overnight.
	if now.Hour() >= 18 {
		return false
	}
	if systemOptionValue(usdExchangeRateLastSourceDateOption) == now.Format(time.DateOnly) {
		return false
	}
	if latest == nil {
		return true
	}
	return now.Unix()-latest.UpdatedAt >= int64(usdExchangeRateRetryInterval.Seconds())
}

func (usdExchangeRateUpdateHandler) Run(ctx context.Context, task *model.SystemTask, runnerID string) {
	loc := shanghaiLocation
	if loc == nil {
		loc = time.FixedZone("CST", 8*60*60)
	}
	sourceDate := time.Now().In(loc).Format(time.DateOnly)
	result, err := refreshUSDExchangeRate(ctx, sourceDate)
	if err != nil {
		finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusFailed, nil, err)
		return
	}
	finishSystemTaskHandler(task, runnerID, model.SystemTaskStatusSucceeded, result, nil)
}

// refreshUSDExchangeRate is shared by the scheduled task and the root-only
// manual refresh endpoint. It never updates Price, the recharge selling price.
func refreshUSDExchangeRate(ctx context.Context, sourceDate string) (usdExchangeRateUpdateResult, error) {
	rate, err := fetchSafeUSDExchangeRate(ctx, sourceDate)
	if err != nil {
		return usdExchangeRateUpdateResult{}, err
	}

	oldRate := systemOptionFloat("USDExchangeRate", 0)
	changed := math.Abs(oldRate-rate) > 0.0000001
	values := map[string]string{
		usdExchangeRateLastSourceOption:     usdExchangeRateSource,
		usdExchangeRateLastSourceDateOption: sourceDate,
		usdExchangeRateLastUpdatedAtOption:  strconv.FormatInt(time.Now().Unix(), 10),
	}
	if changed {
		values["USDExchangeRate"] = strconv.FormatFloat(rate, 'f', 4, 64)
	}
	if err := model.UpdateOptionsBulk(values); err != nil {
		return usdExchangeRateUpdateResult{}, err
	}

	return usdExchangeRateUpdateResult{
		Source:     usdExchangeRateSource,
		SourceDate: sourceDate,
		OldRate:    oldRate,
		NewRate:    rate,
		Changed:    changed,
	}, nil
}

type usdExchangeRateUpdateResult struct {
	Source     string  `json:"source"`
	SourceDate string  `json:"source_date"`
	OldRate    float64 `json:"old_rate"`
	NewRate    float64 `json:"new_rate"`
	Changed    bool    `json:"changed"`
}

func systemOptionValue(key string) string {
	common.OptionMapRWMutex.RLock()
	defer common.OptionMapRWMutex.RUnlock()
	return common.OptionMap[key]
}

func systemOptionIsTrue(key string) bool {
	return strings.EqualFold(strings.TrimSpace(systemOptionValue(key)), "true")
}

func systemOptionFloat(key string, fallback float64) float64 {
	value, err := strconv.ParseFloat(systemOptionValue(key), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return fallback
	}
	return value
}

func fetchSafeUSDExchangeRateForDate(ctx context.Context, date string) (float64, error) {
	requestURL := "https://www.safe.gov.cn/AppStructured/hlw/RMBQuery.do?startDate=" + date + "&endDate=" + date
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("User-Agent", "new-api exchange-rate updater")
	resp, err := safeUSDExchangeRateHTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("SAFE returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return 0, err
	}
	return parseSafeUSDCNYCentralParity(string(body), date)
}

// parseSafeUSDCNYCentralParity extracts the first currency cell (USD) from
// the row for date. SAFE quotes it as CNY per 100 USD, so the stored setting is
// the quoted value divided by 100.
func parseSafeUSDCNYCentralParity(page, date string) (float64, error) {
	quotedDate := regexp.QuoteMeta(date)
	rowPattern := `(?s)<td[^>]*>\s*` + quotedDate + `\s*</td>\s*<td[^>]*>\s*([0-9]+(?:\.[0-9]+)?)\s*</td>`
	matches := regexp.MustCompile(rowPattern).FindStringSubmatch(page)
	if len(matches) != 2 {
		return 0, fmt.Errorf("SAFE response has no USD central parity for %s", date)
	}
	perHundredUSD, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("parse SAFE USD central parity: %w", err)
	}
	rate := perHundredUSD / 100
	if rate < 5 || rate > 10 || math.IsNaN(rate) || math.IsInf(rate, 0) {
		return 0, fmt.Errorf("SAFE USD/CNY central parity %.6f is outside the accepted range", rate)
	}
	return rate, nil
}
