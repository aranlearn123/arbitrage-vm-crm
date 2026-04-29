package handler

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"arbitrage-vm-crm-backend/internal/config"
	"arbitrage-vm-crm-backend/internal/repo"
	"arbitrage-vm-crm-backend/internal/response"
	"github.com/uptrace/bun"
)

const (
	apiVersion                 = "0.1.0"
	systemDBPingTimeout        = 3 * time.Second
	systemMarketDataStaleAfter = 10 * time.Minute
)

type SystemHandler struct {
	db        *bun.DB
	repo      *repo.SystemRepo
	exchanges []systemExchangeConfig
}

type systemExchangeConfig struct {
	Name                 string
	Enabled              bool
	Supported            bool
	CredentialConfigured bool
	Demo                 bool
}

func NewSystemHandler(db *bun.DB, repo *repo.SystemRepo, cfg config.Config) *SystemHandler {
	return &SystemHandler{
		db:   db,
		repo: repo,
		exchanges: []systemExchangeConfig{
			{
				Name:                 "bitget",
				Enabled:              true,
				Supported:            true,
				CredentialConfigured: strings.TrimSpace(cfg.BitgetCredential.APIKey) != "" && strings.TrimSpace(cfg.BitgetCredential.APISecret) != "" && strings.TrimSpace(cfg.BitgetCredential.Passphrase) != "",
				Demo:                 cfg.BitgetDemo,
			},
			{
				Name:                 "bybit",
				Enabled:              true,
				Supported:            true,
				CredentialConfigured: strings.TrimSpace(cfg.BybitCredential.APIKey) != "" && strings.TrimSpace(cfg.BybitCredential.APISecret) != "",
				Demo:                 cfg.BybitDemo,
			},
		},
	}
}

// Status godoc
// @Summary System status
// @Description Get API, database, TimescaleDB, market-data, allocation, and exchange status
// @Tags System
// @Produce json
// @Success 200 {object} response.SystemStatus
// @Failure 503 {object} response.SystemStatus
// @Router /system/status [get]
func (h *SystemHandler) Status(c *Context) error {
	now := time.Now().UTC()
	payload := response.SystemStatus{
		Status:     "ok",
		Time:       now.Format(time.RFC3339),
		APIVersion: apiVersion,
		Database: response.HealthComponent{
			Status: "ok",
		},
		Timescale: response.HealthComponent{
			Status: "ok",
		},
	}

	if err := h.pingDatabase(c.UserContext()); err != nil {
		payload.Status = "unhealthy"
		payload.Database = response.HealthComponent{Status: "unhealthy", Error: err.Error()}
		return c.Status(http.StatusServiceUnavailable).JSON(payload)
	}

	overview, err := h.repo.Overview(c.UserContext())
	if err != nil {
		payload.Status = "unhealthy"
		payload.Database = response.HealthComponent{Status: "unhealthy", Error: err.Error()}
		return c.Status(http.StatusServiceUnavailable).JSON(payload)
	}
	exchanges, err := h.exchangeStatuses(c.UserContext(), now)
	if err != nil {
		payload.Status = "unhealthy"
		payload.Database = response.HealthComponent{Status: "unhealthy", Error: err.Error()}
		return c.Status(http.StatusServiceUnavailable).JSON(payload)
	}

	if !overview.TimescaleEnabled {
		payload.Status = "degraded"
		payload.Timescale = response.HealthComponent{Status: "unavailable", Error: "timescaledb extension is not installed"}
	}

	payload.MarketData = response.SystemMarketDataStatus{
		FundingLastAt:       formatNullTime(overview.FundingLastAt),
		OpenInterestLastAt:  formatNullTime(overview.OpenInterestLastAt),
		MarketQualityLastAt: formatNullTime(overview.MarketQualityLastAt),
	}
	payload.Allocations = response.SystemAllocationStatus{
		Total:         overview.AllocationCount,
		Running:       overview.RunningAllocationCount,
		LastUpdatedAt: formatNullTime(overview.AllocationLastUpdated),
	}
	payload.Exchanges = exchanges

	for _, exchange := range exchanges {
		if exchange.Status != "ok" {
			payload.Status = "degraded"
			break
		}
	}

	return c.JSON(payload)
}

// Exchanges godoc
// @Summary System exchanges
// @Description List supported exchanges and latest market-data timestamps
// @Tags System
// @Produce json
// @Success 200 {object} response.SystemExchangeList
// @Failure 503 {object} response.Error
// @Router /system/exchanges [get]
func (h *SystemHandler) Exchanges(c *Context) error {
	if err := h.pingDatabase(c.UserContext()); err != nil {
		return c.Status(http.StatusServiceUnavailable).JSON(response.Error{Error: err.Error()})
	}

	rows, err := h.exchangeStatuses(c.UserContext(), time.Now().UTC())
	if err != nil {
		return c.Status(http.StatusServiceUnavailable).JSON(response.Error{Error: err.Error()})
	}

	return c.JSON(response.SystemExchangeList{
		Data:  rows,
		Count: len(rows),
	})
}

func (h *SystemHandler) pingDatabase(ctx context.Context) error {
	if h.db == nil {
		return sql.ErrConnDone
	}
	ctx, cancel := context.WithTimeout(ctx, systemDBPingTimeout)
	defer cancel()
	return h.db.PingContext(ctx)
}

func (h *SystemHandler) exchangeStatuses(ctx context.Context, now time.Time) ([]response.SystemExchange, error) {
	rows, err := h.repo.Exchanges(ctx)
	if err != nil {
		return nil, err
	}

	byExchange := make(map[string]repo.SystemExchangeOverview, len(rows))
	for _, row := range rows {
		byExchange[strings.ToLower(row.Exchange)] = row
	}

	out := make([]response.SystemExchange, 0, len(h.exchanges))
	for _, cfg := range h.exchanges {
		row := byExchange[cfg.Name]
		item := response.SystemExchange{
			Exchange:             cfg.Name,
			Enabled:              cfg.Enabled,
			Supported:            cfg.Supported,
			CredentialConfigured: cfg.CredentialConfigured,
			Demo:                 cfg.Demo,
			LastFundingAt:        formatNullTime(row.LastFundingAt),
			LastOpenInterestAt:   formatNullTime(row.LastOpenInterestAt),
			LastMarketQualityAt:  formatNullTime(row.LastMarketQualityAt),
		}

		lastMarketDataAt := maxNullTime(row.LastFundingAt, row.LastOpenInterestAt, row.LastMarketQualityAt)
		item.LastMarketDataAt = formatNullTime(lastMarketDataAt)
		item.Status, item.Notes = exchangeStatus(now, lastMarketDataAt, cfg.CredentialConfigured)
		out = append(out, item)
	}

	return out, nil
}

func exchangeStatus(now time.Time, lastMarketDataAt sql.NullTime, credentialConfigured bool) (string, []string) {
	notes := make([]string, 0, 2)
	if !credentialConfigured {
		notes = append(notes, "credential_not_configured")
	}
	if !lastMarketDataAt.Valid {
		notes = append(notes, "market_data_missing")
		return "no_data", notes
	}
	if now.Sub(lastMarketDataAt.Time.UTC()) > systemMarketDataStaleAfter {
		notes = append(notes, "market_data_stale")
		return "stale", notes
	}
	if len(notes) > 0 {
		return "degraded", notes
	}
	return "ok", nil
}

func formatNullTime(value sql.NullTime) string {
	if !value.Valid || value.Time.IsZero() {
		return ""
	}
	return value.Time.UTC().Format(time.RFC3339)
}

func maxNullTime(values ...sql.NullTime) sql.NullTime {
	var out sql.NullTime
	for _, value := range values {
		if !value.Valid {
			continue
		}
		if !out.Valid || value.Time.After(out.Time) {
			out = value
		}
	}
	return out
}
