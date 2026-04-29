package handler

import (
	"context"
	"strings"

	"arbitrage-vm-crm-backend/internal/repo"
	"arbitrage-vm-crm-backend/internal/response"

	"github.com/gofiber/fiber/v2"
)

type PnLHandler struct {
	repo *repo.PnLRepo
}

func NewPnLHandler(repo *repo.PnLRepo) *PnLHandler {
	return &PnLHandler{repo: repo}
}

// Events godoc
// @Summary PnL events
// @Description List persisted PnL ledger events
// @Tags PnL
// @Produce json
// @Param exchange query string false "Filter by exchange. Supports comma-separated values."
// @Param base query string false "Filter by base asset. Supports comma-separated values."
// @Param quote query string false "Filter by quote asset. Supports comma-separated values."
// @Param component query string false "Filter by component: funding, trading_fee, trading_pnl. Supports comma-separated values."
// @Param source_type query string false "Filter by source type. Supports comma-separated values."
// @Param source_id query string false "Filter by source id. Supports comma-separated values."
// @Param from query string false "Inclusive start time. Supports RFC3339 or YYYY-MM-DD."
// @Param to query string false "Exclusive end time. Supports RFC3339 or YYYY-MM-DD."
// @Param limit query int false "Maximum rows to return" default(100)
// @Success 200 {object} response.PnLEventList
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /pnl/events [get]
func (h *PnLHandler) Events(c *fiber.Ctx) error {
	filter, err := pnlFilter(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error{Error: err.Error()})
	}

	rows, err := h.repo.Events(c.UserContext(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error{Error: err.Error()})
	}

	return c.JSON(response.PnLEventList{
		Data:  mapPnLEvents(rows),
		Count: len(rows),
		Limit: filter.Limit,
	})
}

// Summary godoc
// @Summary PnL summary
// @Description Summarize PnL totals and grouped breakdowns
// @Tags PnL
// @Produce json
// @Param exchange query string false "Filter by exchange. Supports comma-separated values."
// @Param base query string false "Filter by base asset. Supports comma-separated values."
// @Param quote query string false "Filter by quote asset. Supports comma-separated values."
// @Param component query string false "Filter by component: funding, trading_fee, trading_pnl. Supports comma-separated values."
// @Param source_type query string false "Filter by source type. Supports comma-separated values."
// @Param source_id query string false "Filter by source id. Supports comma-separated values."
// @Param from query string false "Inclusive start time. Supports RFC3339 or YYYY-MM-DD."
// @Param to query string false "Exclusive end time. Supports RFC3339 or YYYY-MM-DD."
// @Param limit query int false "Maximum grouped rows per breakdown" default(100)
// @Success 200 {object} response.PnLSummary
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /pnl/summary [get]
func (h *PnLHandler) Summary(c *fiber.Ctx) error {
	filter, err := pnlFilter(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error{Error: err.Error()})
	}

	summary, err := h.repo.Summary(c.UserContext(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error{Error: err.Error()})
	}
	byComponent, err := h.repo.ByComponent(c.UserContext(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error{Error: err.Error()})
	}
	byExchange, err := h.repo.ByExchange(c.UserContext(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error{Error: err.Error()})
	}
	byPair, err := h.repo.ByPair(c.UserContext(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error{Error: err.Error()})
	}

	return c.JSON(response.PnLSummary{
		Count:            summary.Count,
		TotalAmount:      summary.TotalAmount,
		FundingAmount:    summary.FundingAmount,
		TradingFeeAmount: summary.TradingFeeAmount,
		TradingPnLAmount: summary.TradingPnLAmount,
		ByComponent:      mapPnLGroups(byComponent),
		ByExchange:       mapPnLGroups(byExchange),
		ByPair:           mapPnLGroups(byPair),
	})
}

// ByPair godoc
// @Summary PnL by pair
// @Description Summarize PnL grouped by base and quote
// @Tags PnL
// @Produce json
// @Param exchange query string false "Filter by exchange. Supports comma-separated values."
// @Param base query string false "Filter by base asset. Supports comma-separated values."
// @Param quote query string false "Filter by quote asset. Supports comma-separated values."
// @Param component query string false "Filter by component: funding, trading_fee, trading_pnl. Supports comma-separated values."
// @Param source_type query string false "Filter by source type. Supports comma-separated values."
// @Param source_id query string false "Filter by source id. Supports comma-separated values."
// @Param from query string false "Inclusive start time. Supports RFC3339 or YYYY-MM-DD."
// @Param to query string false "Exclusive end time. Supports RFC3339 or YYYY-MM-DD."
// @Param limit query int false "Maximum rows to return" default(100)
// @Success 200 {object} response.PnLGroupList
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /pnl/by-pair [get]
func (h *PnLHandler) ByPair(c *fiber.Ctx) error {
	return h.group(c, h.repo.ByPair)
}

// ByExchange godoc
// @Summary PnL by exchange
// @Description Summarize PnL grouped by exchange
// @Tags PnL
// @Produce json
// @Param exchange query string false "Filter by exchange. Supports comma-separated values."
// @Param base query string false "Filter by base asset. Supports comma-separated values."
// @Param quote query string false "Filter by quote asset. Supports comma-separated values."
// @Param component query string false "Filter by component: funding, trading_fee, trading_pnl. Supports comma-separated values."
// @Param source_type query string false "Filter by source type. Supports comma-separated values."
// @Param source_id query string false "Filter by source id. Supports comma-separated values."
// @Param from query string false "Inclusive start time. Supports RFC3339 or YYYY-MM-DD."
// @Param to query string false "Exclusive end time. Supports RFC3339 or YYYY-MM-DD."
// @Param limit query int false "Maximum rows to return" default(100)
// @Success 200 {object} response.PnLGroupList
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /pnl/by-exchange [get]
func (h *PnLHandler) ByExchange(c *fiber.Ctx) error {
	return h.group(c, h.repo.ByExchange)
}

// ByComponent godoc
// @Summary PnL by component
// @Description Summarize PnL grouped by funding, trading_fee, and trading_pnl
// @Tags PnL
// @Produce json
// @Param exchange query string false "Filter by exchange. Supports comma-separated values."
// @Param base query string false "Filter by base asset. Supports comma-separated values."
// @Param quote query string false "Filter by quote asset. Supports comma-separated values."
// @Param component query string false "Filter by component: funding, trading_fee, trading_pnl. Supports comma-separated values."
// @Param source_type query string false "Filter by source type. Supports comma-separated values."
// @Param source_id query string false "Filter by source id. Supports comma-separated values."
// @Param from query string false "Inclusive start time. Supports RFC3339 or YYYY-MM-DD."
// @Param to query string false "Exclusive end time. Supports RFC3339 or YYYY-MM-DD."
// @Param limit query int false "Maximum rows to return" default(100)
// @Success 200 {object} response.PnLGroupList
// @Failure 400 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /pnl/by-component [get]
func (h *PnLHandler) ByComponent(c *fiber.Ctx) error {
	return h.group(c, h.repo.ByComponent)
}

func (h *PnLHandler) group(c *fiber.Ctx, load func(context.Context, repo.PnLFilter) ([]repo.PnLGroup, error)) error {
	filter, err := pnlFilter(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error{Error: err.Error()})
	}

	rows, err := load(c.UserContext(), filter)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error{Error: err.Error()})
	}

	return c.JSON(response.PnLGroupList{
		Data:  mapPnLGroups(rows),
		Count: len(rows),
		Limit: filter.Limit,
	})
}

func pnlFilter(c *fiber.Ctx) (repo.PnLFilter, error) {
	from, err := queryTime(c, "from")
	if err != nil {
		return repo.PnLFilter{}, err
	}
	to, err := queryTime(c, "to")
	if err != nil {
		return repo.PnLFilter{}, err
	}

	return repo.PnLFilter{
		Exchanges:   queryCSV(c.Query("exchange"), strings.ToLower),
		Bases:       queryCSV(c.Query("base"), strings.ToUpper),
		Quotes:      queryCSV(c.Query("quote"), strings.ToUpper),
		Components:  queryCSV(c.Query("component"), strings.ToLower),
		SourceTypes: queryCSV(c.Query("source_type"), strings.ToLower),
		SourceIDs:   queryCSV(c.Query("source_id"), strings.TrimSpace),
		From:        from,
		To:          to,
		Limit:       queryLimit(c),
	}, nil
}

func mapPnLEvents(rows []repo.PnLEvent) []response.PnLEvent {
	out := make([]response.PnLEvent, 0, len(rows))
	for _, row := range rows {
		out = append(out, response.PnLEvent{
			EventTime:  formatTime(row.EventTime),
			Exchange:   row.Exchange,
			Base:       row.Base,
			Quote:      row.Quote,
			Pair:       row.Base + row.Quote,
			Component:  row.Component,
			Amount:     row.Amount,
			Currency:   row.Currency,
			SourceType: row.SourceType,
			SourceID:   row.SourceID,
			CreatedAt:  formatTime(row.CreatedAt),
		})
	}
	return out
}

func mapPnLGroups(rows []repo.PnLGroup) []response.PnLGroup {
	out := make([]response.PnLGroup, 0, len(rows))
	for _, row := range rows {
		pair := ""
		if row.Base != "" || row.Quote != "" {
			pair = row.Base + row.Quote
		}
		out = append(out, response.PnLGroup{
			Exchange:         row.Exchange,
			Base:             row.Base,
			Quote:            row.Quote,
			Pair:             pair,
			Component:        row.Component,
			Count:            row.Count,
			TotalAmount:      row.TotalAmount,
			FundingAmount:    row.FundingAmount,
			TradingFeeAmount: row.TradingFeeAmount,
			TradingPnLAmount: row.TradingPnLAmount,
			FirstEventTime:   formatTime(row.FirstEventTime),
			LastEventTime:    formatTime(row.LastEventTime),
		})
	}
	return out
}
