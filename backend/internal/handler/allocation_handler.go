package handler

import (
	"strconv"
	"time"

	"arbitrage-vm-crm-backend/internal/repo"
	"arbitrage-vm-crm-backend/internal/response"

	"github.com/gofiber/fiber/v2"
)

const defaultHandlerLimit = 100

type AllocationHandler struct {
	repo *repo.AllocationRepo
}

func NewAllocationHandler(repo *repo.AllocationRepo) *AllocationHandler {
	return &AllocationHandler{repo: repo}
}

// Summary godoc
// @Summary Allocation summary
// @Description Count allocations by status and show active budget
// @Tags Allocations
// @Produce json
// @Success 200 {object} response.AllocationSummary
// @Failure 500 {object} response.Error
// @Router /allocations/summary [get]
func (h *AllocationHandler) Summary(c *fiber.Ctx) error {
	summary, err := h.repo.Summary(c.UserContext())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error{Error: err.Error()})
	}

	reasons, err := h.repo.CancelledReasons(c.UserContext(), 10)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error{Error: err.Error()})
	}

	return c.JSON(response.AllocationSummary{
		Total:             summary.Total,
		Active:            summary.Active,
		Cancelled:         summary.Cancelled,
		ActiveBudgetUSD:   summary.ActiveBudgetUSD,
		ByStatus:          mapStatusCounts(summary.ByStatus),
		CancelledByReason: mapCancelledReasons(reasons),
	})
}

// Active godoc
// @Summary Active allocations
// @Description List allocations in created, running, failed, or paused state
// @Tags Allocations
// @Produce json
// @Param limit query int false "Maximum rows to return" default(100)
// @Success 200 {object} response.AllocationList
// @Failure 500 {object} response.Error
// @Router /allocations/active [get]
func (h *AllocationHandler) Active(c *fiber.Ctx) error {
	limit := queryLimit(c)
	rows, err := h.repo.Active(c.UserContext(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error{Error: err.Error()})
	}
	return c.JSON(response.AllocationList{
		Data:  mapAllocations(rows),
		Count: len(rows),
		Limit: limit,
	})
}

// Running godoc
// @Summary Running allocations
// @Description List allocations currently in running state
// @Tags Allocations
// @Produce json
// @Param limit query int false "Maximum rows to return" default(100)
// @Success 200 {object} response.AllocationList
// @Failure 500 {object} response.Error
// @Router /allocations/running [get]
func (h *AllocationHandler) Running(c *fiber.Ctx) error {
	limit := queryLimit(c)
	rows, err := h.repo.Running(c.UserContext(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error{Error: err.Error()})
	}
	return c.JSON(response.AllocationList{
		Data:  mapAllocations(rows),
		Count: len(rows),
		Limit: limit,
	})
}

// CancelledReasons godoc
// @Summary Cancelled allocation reasons
// @Description Count cancelled allocations grouped by note/reason
// @Tags Allocations
// @Produce json
// @Param limit query int false "Maximum rows to return" default(100)
// @Success 200 {object} response.CancelledReasonList
// @Failure 500 {object} response.Error
// @Router /allocations/cancelled/reasons [get]
func (h *AllocationHandler) CancelledReasons(c *fiber.Ctx) error {
	limit := queryLimit(c)
	rows, err := h.repo.CancelledReasons(c.UserContext(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error{Error: err.Error()})
	}
	return c.JSON(response.CancelledReasonList{
		Data:  mapCancelledReasons(rows),
		Count: len(rows),
		Limit: limit,
	})
}

func queryLimit(c *fiber.Ctx) int {
	limit, err := strconv.Atoi(c.Query("limit"))
	if err != nil {
		return defaultHandlerLimit
	}
	if limit <= 0 {
		return defaultHandlerLimit
	}
	if limit > 500 {
		return 500
	}
	return limit
}

func mapAllocations(rows []repo.Allocation) []response.Allocation {
	out := make([]response.Allocation, 0, len(rows))
	for _, row := range rows {
		out = append(out, response.Allocation{
			ID:        row.ID,
			Base:      row.Base,
			Quote:     row.Quote,
			Pair:      row.Base + row.Quote,
			Direction: row.Direction,
			Rank:      row.Rank,
			Score:     row.Score,
			Role:      row.Role,
			Status:    row.Status,
			BudgetUSD: row.BudgetUSD,
			WorkerPID: row.WorkerPID,
			Note:      row.Note,
			CreatedAt: formatTime(row.CreatedAt),
			UpdatedAt: formatTime(row.UpdatedAt),
		})
	}
	return out
}

func mapStatusCounts(rows []repo.AllocationStatusCount) map[string]int64 {
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		out[row.Status] = row.Count
	}
	return out
}

func mapCancelledReasons(rows []repo.CancelledReasonCount) []response.CancelledReasonCount {
	out := make([]response.CancelledReasonCount, 0, len(rows))
	for _, row := range rows {
		out = append(out, response.CancelledReasonCount{
			Reason: row.Reason,
			Count:  row.Count,
		})
	}
	return out
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
