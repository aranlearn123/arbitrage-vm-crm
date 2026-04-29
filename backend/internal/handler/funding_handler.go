package handler

import (
	"net/http"
	"strings"

	"arbitrage-vm-crm-backend/internal/repo"
	"arbitrage-vm-crm-backend/internal/response"
)

type FundingHandler struct {
	repo *repo.FundingRepo
}

func NewFundingHandler(repo *repo.FundingRepo) *FundingHandler {
	return &FundingHandler{repo: repo}
}

// Latest godoc
// @Summary Latest funding rates
// @Description List latest funding rate per exchange and pair
// @Tags Funding
// @Produce json
// @Param exchange query string false "Filter by exchange. Supports comma-separated values."
// @Param base query string false "Filter by base asset. Supports comma-separated values."
// @Param quote query string false "Filter by quote asset. Supports comma-separated values."
// @Param limit query int false "Maximum rows to return" default(100)
// @Success 200 {object} response.FundingRateList
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /funding/latest [get]
func (h *FundingHandler) Latest(c *Context) error {
	rows, err := h.repo.Latest(c.UserContext(), marketDataFilter(c))
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(response.Error{Error: err.Error()})
	}

	limit := queryLimit(c)
	return c.JSON(response.FundingRateList{
		Data:  mapFundingRates(rows),
		Count: len(rows),
		Limit: limit,
	})
}

// History godoc
// @Summary Funding history
// @Description List historical funding rates
// @Tags Funding
// @Produce json
// @Param exchange query string false "Filter by exchange. Supports comma-separated values."
// @Param base query string false "Filter by base asset. Supports comma-separated values."
// @Param quote query string false "Filter by quote asset. Supports comma-separated values."
// @Param from query string false "Inclusive start time. Supports RFC3339 or YYYY-MM-DD."
// @Param to query string false "Exclusive end time. Supports RFC3339 or YYYY-MM-DD."
// @Param limit query int false "Maximum rows to return" default(100)
// @Success 200 {object} response.FundingRateList
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /funding/history [get]
func (h *FundingHandler) History(c *Context) error {
	filter, err := marketDataHistoryFilter(c)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(response.Error{Error: err.Error()})
	}

	rows, err := h.repo.History(c.UserContext(), filter)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(response.Error{Error: err.Error()})
	}

	return c.JSON(response.FundingRateList{
		Data:  mapFundingRates(rows),
		Count: len(rows),
		Limit: filter.Limit,
	})
}

// Spread godoc
// @Summary Funding spread
// @Description Get latest Bybit versus Bitget funding spread for one pair
// @Tags Funding
// @Produce json
// @Param base query string true "Base asset"
// @Param quote query string false "Quote asset" default(USDT)
// @Success 200 {object} response.FundingSpreadDetail
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /funding/spread [get]
func (h *FundingHandler) Spread(c *Context) error {
	base := strings.ToUpper(strings.TrimSpace(c.Query("base")))
	if base == "" {
		return c.Status(http.StatusBadRequest).JSON(response.Error{Error: "base is required"})
	}
	quote := strings.ToUpper(strings.TrimSpace(c.Query("quote", "USDT")))

	row, err := h.repo.Spread(c.UserContext(), base, quote)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(response.Error{Error: err.Error()})
	}
	if row == nil {
		return c.Status(http.StatusNotFound).JSON(response.Error{Error: "funding spread not found"})
	}

	return c.JSON(response.FundingSpreadDetail{
		Data: mapFundingSpread(*row),
	})
}

// TopSpreads godoc
// @Summary Top funding spreads
// @Description List pairs with the largest absolute Bybit versus Bitget funding spread
// @Tags Funding
// @Produce json
// @Param base query string false "Filter by base asset. Supports comma-separated values."
// @Param quote query string false "Filter by quote asset. Supports comma-separated values."
// @Param min_abs_spread_bps query number false "Minimum absolute spread in bps" default(0)
// @Param limit query int false "Maximum rows to return" default(100)
// @Success 200 {object} response.FundingSpreadList
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /funding/top-spreads [get]
func (h *FundingHandler) TopSpreads(c *Context) error {
	minAbsSpreadBps, err := queryFloat(c, "min_abs_spread_bps", 0)
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(response.Error{Error: err.Error()})
	}

	limit := queryLimit(c)
	rows, err := h.repo.TopSpreads(c.UserContext(), repo.FundingSpreadFilter{
		Bases:           queryCSV(c.Query("base"), strings.ToUpper),
		Quotes:          queryCSV(c.Query("quote"), strings.ToUpper),
		MinAbsSpreadBps: minAbsSpreadBps,
		Limit:           limit,
	})
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(response.Error{Error: err.Error()})
	}

	return c.JSON(response.FundingSpreadList{
		Data:  mapFundingSpreads(rows),
		Count: len(rows),
		Limit: limit,
	})
}

func marketDataFilter(c *Context) repo.TimeSeriesFilter {
	return repo.TimeSeriesFilter{
		Exchanges: queryCSV(c.Query("exchange"), strings.ToLower),
		Bases:     queryCSV(c.Query("base"), strings.ToUpper),
		Quotes:    queryCSV(c.Query("quote"), strings.ToUpper),
		Limit:     queryLimit(c),
	}
}

func marketDataHistoryFilter(c *Context) (repo.TimeSeriesFilter, error) {
	from, err := queryTime(c, "from")
	if err != nil {
		return repo.TimeSeriesFilter{}, err
	}
	to, err := queryTime(c, "to")
	if err != nil {
		return repo.TimeSeriesFilter{}, err
	}

	filter := marketDataFilter(c)
	filter.From = from
	filter.To = to
	return filter, nil
}

func mapFundingRates(rows []repo.Funding) []response.FundingRate {
	out := make([]response.FundingRate, 0, len(rows))
	for _, row := range rows {
		out = append(out, response.FundingRate{
			Time:        formatTime(row.Time),
			Exchange:    row.Exchange,
			Base:        row.Base,
			Quote:       row.Quote,
			Pair:        row.Base + row.Quote,
			FundingRate: row.FundingRate,
		})
	}
	return out
}

func mapFundingSpreads(rows []repo.FundingSpread) []response.FundingSpread {
	out := make([]response.FundingSpread, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapFundingSpread(row))
	}
	return out
}

func mapFundingSpread(row repo.FundingSpread) response.FundingSpread {
	return response.FundingSpread{
		Time:         formatTime(row.Time),
		Base:         row.Base,
		Quote:        row.Quote,
		Pair:         row.Base + row.Quote,
		BybitRate:    row.BybitRate,
		BitgetRate:   row.BitgetRate,
		Spread:       row.Spread,
		SpreadBps:    row.SpreadBps,
		AbsSpreadBps: row.AbsSpreadBps,
		Direction:    row.Direction,
		BybitTime:    formatTime(row.BybitTime),
		BitgetTime:   formatTime(row.BitgetTime),
	}
}
