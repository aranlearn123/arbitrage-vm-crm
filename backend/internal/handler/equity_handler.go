package handler

import (
	"strings"

	"arbitrage-vm-crm-backend/internal/exchange"
	"arbitrage-vm-crm-backend/internal/response"

	"github.com/gofiber/fiber/v2"
)

type EquityHandler struct {
	service *exchange.EquityService
}

func NewEquityHandler(service *exchange.EquityService) *EquityHandler {
	return &EquityHandler{service: service}
}

// Latest godoc
// @Summary Latest equity from exchange API
// @Description Temporarily fetch latest wallet/equity directly from exchange APIs with a short backend cache
// @Tags Equity
// @Produce json
// @Param exchange query string false "Filter by exchange: bybit, bitget. Supports comma-separated values."
// @Param quote query string false "Margin quote coins. Supports comma-separated values." default(USDT)
// @Param refresh query bool false "Bypass backend cache and pull from exchange API"
// @Success 200 {object} response.EquityLatest
// @Failure 400 {object} response.Error
// @Failure 502 {object} response.EquityLatest
// @Router /equity/latest [get]
// @Router /equity/live [get]
func (h *EquityHandler) Latest(c *fiber.Ctx) error {
	refresh := strings.EqualFold(strings.TrimSpace(c.Query("refresh")), "true")
	result, err := h.service.Fetch(c.UserContext(),
		queryCSV(c.Query("exchange"), strings.ToLower),
		queryCSV(c.Query("quote"), strings.ToUpper),
		refresh,
	)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error{Error: err.Error()})
	}

	status := fiber.StatusOK
	if len(result.Snapshots) == 0 && len(result.Errors) > 0 {
		status = fiber.StatusBadGateway
	}

	return c.Status(status).JSON(response.EquityLatest{
		Data:            mapEquitySnapshots(result.Snapshots),
		Combined:        mapCombinedEquity(result.Combined),
		Errors:          mapEquityErrors(result.Errors),
		Count:           len(result.Snapshots),
		CacheTTLSeconds: int(result.CacheTTL.Seconds()),
	})
}

func mapEquitySnapshots(rows []exchange.WalletSnapshot) []response.EquitySnapshot {
	out := make([]response.EquitySnapshot, 0, len(rows))
	for _, row := range rows {
		out = append(out, response.EquitySnapshot{
			Exchange:          row.Exchange,
			Time:              formatTime(row.Time),
			AccountEquity:     row.AccountEquity,
			WalletBalance:     row.WalletBalance,
			AvailableBalance:  row.AvailableBalance,
			UnrealizedPnL:     row.UnrealizedPnL,
			InitialMargin:     row.InitialMargin,
			MaintenanceMargin: row.MaintenanceMargin,
			Coins:             mapEquityCoins(row.Coins),
			Source:            row.Source,
			Cached:            row.Cached,
		})
	}
	return out
}

func mapEquityCoins(rows []exchange.CoinBalance) []response.EquityCoin {
	out := make([]response.EquityCoin, 0, len(rows))
	for _, row := range rows {
		out = append(out, response.EquityCoin{
			Coin:             row.Coin,
			Equity:           row.Equity,
			WalletBalance:    row.WalletBalance,
			AvailableBalance: row.AvailableBalance,
			Locked:           row.Locked,
			UnrealizedPnL:    row.UnrealizedPnL,
			USDValue:         row.USDValue,
		})
	}
	return out
}

func mapCombinedEquity(row exchange.CombinedEquity) response.CombinedEquity {
	return response.CombinedEquity{
		Time:              formatTime(row.Time),
		AccountEquity:     row.AccountEquity,
		WalletBalance:     row.WalletBalance,
		AvailableBalance:  row.AvailableBalance,
		UnrealizedPnL:     row.UnrealizedPnL,
		InitialMargin:     row.InitialMargin,
		MaintenanceMargin: row.MaintenanceMargin,
	}
}

func mapEquityErrors(rows []exchange.FetchError) []response.EquityFetchError {
	out := make([]response.EquityFetchError, 0, len(rows))
	for _, row := range rows {
		out = append(out, response.EquityFetchError{
			Exchange: row.Exchange,
			Error:    row.Message,
		})
	}
	return out
}
