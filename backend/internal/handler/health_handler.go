package handler

import (
	"context"
	"net/http"
	"time"

	"arbitrage-vm-crm-backend/internal/response"
	"github.com/uptrace/bun"
)

const healthDBPingTimeout = 3 * time.Second

type HealthHandler struct {
	db *bun.DB
}

func NewHealthHandler(db *bun.DB) *HealthHandler {
	return &HealthHandler{
		db: db,
	}
}

// Check godoc
// @Summary Health check
// @Description Check API service and database status
// @Tags System
// @Produce json
// @Success 200 {object} response.Health
// @Failure 503 {object} response.Health
// @Router /health [get]
func (h *HealthHandler) Check(c *Context) error {
	payload := response.Health{
		Status: "ok",
		Time:   time.Now().UTC().Format(time.RFC3339),
		Checks: response.HealthChecks{
			API: response.HealthComponent{
				Status: "ok",
			},
			Database: response.HealthComponent{
				Status: "ok",
			},
		},
	}

	if h.db == nil {
		payload.Status = "unhealthy"
		payload.Checks.Database = response.HealthComponent{
			Status: "unhealthy",
			Error:  "database is not configured",
		}
		return c.Status(http.StatusServiceUnavailable).JSON(payload)
	}

	ctx, cancel := context.WithTimeout(c.UserContext(), healthDBPingTimeout)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		payload.Status = "unhealthy"
		payload.Checks.Database = response.HealthComponent{
			Status: "unhealthy",
			Error:  err.Error(),
		}
		return c.Status(http.StatusServiceUnavailable).JSON(payload)
	}

	return c.JSON(payload)
}
