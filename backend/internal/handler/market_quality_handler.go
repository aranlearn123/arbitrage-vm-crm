package handler

import (
	"net/http"
	"strconv"
	"strings"

	"arbitrage-vm-crm-backend/internal/repo"
	"arbitrage-vm-crm-backend/internal/response"
)

type MarketQualityHandler struct {
	repo *repo.MarketQualityRepo
}

func NewMarketQualityHandler(repo *repo.MarketQualityRepo) *MarketQualityHandler {
	return &MarketQualityHandler{repo: repo}
}

// Latest godoc
// @Summary Latest market quality metrics
// @Description List latest market quality metrics per exchange and pair
// @Tags Market Quality
// @Produce json
// @Param exchange query string false "Filter by exchange. Supports comma-separated values."
// @Param base query string false "Filter by base asset. Supports comma-separated values."
// @Param quote query string false "Filter by quote asset. Supports comma-separated values."
// @Param limit query int false "Maximum rows to return" default(100)
// @Success 200 {object} response.MarketQualityList
// @Failure 500 {object} response.Error
// @Router /market-quality/latest [get]
func (h *MarketQualityHandler) Latest(c *Context) error {
	filter := marketDataFilter(c)
	rows, err := h.repo.Latest(c.UserContext(), filter)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(response.Error{Error: err.Error()})
	}

	return c.JSON(response.MarketQualityList{
		Data:  mapMarketQualityMetrics(rows),
		Count: len(rows),
		Limit: filter.Limit,
	})
}

// History godoc
// @Summary Market quality history
// @Description List historical market quality metrics
// @Tags Market Quality
// @Produce json
// @Param exchange query string false "Filter by exchange. Supports comma-separated values."
// @Param base query string false "Filter by base asset. Supports comma-separated values."
// @Param quote query string false "Filter by quote asset. Supports comma-separated values."
// @Param from query string false "Inclusive start time. Supports RFC3339 or YYYY-MM-DD."
// @Param to query string false "Exclusive end time. Supports RFC3339 or YYYY-MM-DD."
// @Param limit query int false "Maximum rows to return" default(100)
// @Success 200 {object} response.MarketQualityList
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /market-quality/history [get]
func (h *MarketQualityHandler) History(c *Context) error {
	filter, err := marketDataHistoryFilter(c)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(response.Error{Error: err.Error()})
	}

	rows, err := h.repo.History(c.UserContext(), filter)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(response.Error{Error: err.Error()})
	}

	return c.JSON(response.MarketQualityList{
		Data:  mapMarketQualityMetrics(rows),
		Count: len(rows),
		Limit: filter.Limit,
	})
}

// Alerts godoc
// @Summary Market quality alerts
// @Description List latest market quality rows that violate simple CRM thresholds
// @Tags Market Quality
// @Produce json
// @Param exchange query string false "Filter by exchange. Supports comma-separated values."
// @Param base query string false "Filter by base asset. Supports comma-separated values."
// @Param quote query string false "Filter by quote asset. Supports comma-separated values."
// @Param min_samples query int false "Alert when samples are below this value" default(10)
// @Param max_spread_bps_p50 query number false "Alert when spread_bps_p50 is above this value" default(10)
// @Param max_mid_speed_bps_per_sec_p95 query number false "Alert when mid speed p95 is above this value" default(50)
// @Param min_depth_stability_ratio query number false "Alert when depth stability ratio is below this value" default(0.10)
// @Param limit query int false "Maximum rows to return" default(100)
// @Success 200 {object} response.MarketQualityAlertList
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /market-quality/alerts [get]
func (h *MarketQualityHandler) Alerts(c *Context) error {
	filter, err := marketQualityAlertFilter(c)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(response.Error{Error: err.Error()})
	}

	rows, err := h.repo.Alerts(c.UserContext(), filter)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(response.Error{Error: err.Error()})
	}

	return c.JSON(response.MarketQualityAlertList{
		Data:  mapMarketQualityAlerts(rows, filter),
		Count: len(rows),
		Limit: filter.Limit,
	})
}

func marketQualityAlertFilter(c *Context) (repo.MarketQualityAlertFilter, error) {
	minSamples, err := queryInt(c, "min_samples", 10)
	if err != nil {
		return repo.MarketQualityAlertFilter{}, err
	}
	maxSpreadBpsP50, err := queryFloat(c, "max_spread_bps_p50", 10)
	if err != nil {
		return repo.MarketQualityAlertFilter{}, err
	}
	maxMidSpeedP95, err := queryFloat(c, "max_mid_speed_bps_per_sec_p95", 50)
	if err != nil {
		return repo.MarketQualityAlertFilter{}, err
	}
	minDepthStability, err := queryFloat(c, "min_depth_stability_ratio", 0.10)
	if err != nil {
		return repo.MarketQualityAlertFilter{}, err
	}

	return repo.MarketQualityAlertFilter{
		TimeSeriesFilter:        marketDataFilter(c),
		MinSamples:              minSamples,
		MaxSpreadBpsP50:         maxSpreadBpsP50,
		MaxMidSpeedBpsPerSecP95: maxMidSpeedP95,
		MinDepthStabilityRatio:  minDepthStability,
	}, nil
}

func mapMarketQualityMetrics(rows []repo.MarketQualityMetric) []response.MarketQualityMetric {
	out := make([]response.MarketQualityMetric, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapMarketQualityMetric(row))
	}
	return out
}

func mapMarketQualityMetric(row repo.MarketQualityMetric) response.MarketQualityMetric {
	return response.MarketQualityMetric{
		Time:                 formatTime(row.Time),
		Exchange:             row.Exchange,
		Base:                 row.Base,
		Quote:                row.Quote,
		Pair:                 row.Base + row.Quote,
		Samples:              row.Samples,
		SpreadBpsP50:         row.SpreadBpsP50,
		MidSpeedBpsPerSecP95: row.MidSpeedBpsPerSecP95,
		DepthStabilityRatio:  row.DepthStabilityRatio,
	}
}

func mapMarketQualityAlerts(rows []repo.MarketQualityMetric, filter repo.MarketQualityAlertFilter) []response.MarketQualityAlert {
	out := make([]response.MarketQualityAlert, 0, len(rows))
	for _, row := range rows {
		out = append(out, response.MarketQualityAlert{
			MarketQualityMetric: mapMarketQualityMetric(row),
			Reasons:             marketQualityAlertReasons(row, filter),
		})
	}
	return out
}

func marketQualityAlertReasons(row repo.MarketQualityMetric, filter repo.MarketQualityAlertFilter) []string {
	reasons := make([]string, 0, 4)
	if row.Samples < filter.MinSamples {
		reasons = append(reasons, "low_samples")
	}
	if metricGreaterThan(row.SpreadBpsP50, filter.MaxSpreadBpsP50) {
		reasons = append(reasons, "wide_spread")
	}
	if metricGreaterThan(row.MidSpeedBpsPerSecP95, filter.MaxMidSpeedBpsPerSecP95) {
		reasons = append(reasons, "fast_mid_price")
	}
	if metricLessThan(row.DepthStabilityRatio, filter.MinDepthStabilityRatio) {
		reasons = append(reasons, "unstable_depth")
	}
	return reasons
}

func metricGreaterThan(value string, threshold float64) bool {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return err == nil && parsed > threshold
}

func metricLessThan(value string, threshold float64) bool {
	parsed, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
	return err == nil && parsed < threshold
}
