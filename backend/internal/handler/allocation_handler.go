package handler

import (
	"strconv"
	"strings"
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

// List godoc
// @Summary List allocations
// @Description List allocations with optional status, base, quote, and role filters
// @Tags Allocations
// @Produce json
// @Param status query string false "Filter by status. Supports comma-separated values."
// @Param base query string false "Filter by base asset. Supports comma-separated values."
// @Param quote query string false "Filter by quote asset. Supports comma-separated values."
// @Param role query string false "Filter by crowding role. Supports comma-separated values."
// @Param limit query int false "Maximum rows to return" default(100)
// @Success 200 {object} response.AllocationList
// @Failure 500 {object} response.Error
// @Router /allocations [get]
func (h *AllocationHandler) List(c *fiber.Ctx) error {
	limit := queryLimit(c)
	rows, err := h.repo.List(c.UserContext(), repo.AllocationListFilter{
		Statuses: queryCSV(c.Query("status"), strings.ToLower),
		Bases:    queryCSV(c.Query("base"), strings.ToUpper),
		Quotes:   queryCSV(c.Query("quote"), strings.ToUpper),
		Roles:    queryCSV(c.Query("role"), strings.ToLower),
		Limit:    limit,
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error{Error: err.Error()})
	}
	return c.JSON(response.AllocationList{
		Data:  mapAllocations(rows),
		Count: len(rows),
		Limit: limit,
	})
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

// Cancelled godoc
// @Summary Cancelled allocations
// @Description List cancelled allocations with optional base, quote, role, and reason filters
// @Tags Allocations
// @Produce json
// @Param base query string false "Filter by base asset. Supports comma-separated values."
// @Param quote query string false "Filter by quote asset. Supports comma-separated values."
// @Param role query string false "Filter by crowding role. Supports comma-separated values."
// @Param reason query string false "Filter by normalized cancel reason. Supports comma-separated values."
// @Param limit query int false "Maximum rows to return" default(100)
// @Success 200 {object} response.AllocationList
// @Failure 500 {object} response.Error
// @Router /allocations/cancelled [get]
func (h *AllocationHandler) Cancelled(c *fiber.Ctx) error {
	limit := queryLimit(c)
	rows, err := h.repo.Cancelled(c.UserContext(), repo.AllocationListFilter{
		Bases:   queryCSV(c.Query("base"), strings.ToUpper),
		Quotes:  queryCSV(c.Query("quote"), strings.ToUpper),
		Roles:   queryCSV(c.Query("role"), strings.ToLower),
		Reasons: queryCSV(c.Query("reason"), strings.TrimSpace),
		Limit:   limit,
	})
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

// Detail godoc
// @Summary Allocation detail
// @Description Get one allocation by id
// @Tags Allocations
// @Produce json
// @Param id path int true "Allocation ID"
// @Success 200 {object} response.AllocationDetail
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /allocations/{id} [get]
func (h *AllocationHandler) Detail(c *fiber.Ctx) error {
	id, ok := pathID(c)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error{Error: "invalid allocation id"})
	}

	row, err := h.repo.GetByID(c.UserContext(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error{Error: err.Error()})
	}
	if row == nil {
		return c.Status(fiber.StatusNotFound).JSON(response.Error{Error: "allocation not found"})
	}

	return c.JSON(response.AllocationDetail{
		Data: mapAllocation(*row),
	})
}

// Timeline godoc
// @Summary Allocation timeline
// @Description Build allocation timeline from allocation, scaling, recovery, routing, and order progress tables
// @Tags Allocations
// @Produce json
// @Param id path int true "Allocation ID"
// @Success 200 {object} response.AllocationTimeline
// @Failure 400 {object} response.Error
// @Failure 404 {object} response.Error
// @Failure 500 {object} response.Error
// @Router /allocations/{id}/timeline [get]
func (h *AllocationHandler) Timeline(c *fiber.Ctx) error {
	id, ok := pathID(c)
	if !ok {
		return c.Status(fiber.StatusBadRequest).JSON(response.Error{Error: "invalid allocation id"})
	}

	row, err := h.repo.GetByID(c.UserContext(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error{Error: err.Error()})
	}
	if row == nil {
		return c.Status(fiber.StatusNotFound).JSON(response.Error{Error: "allocation not found"})
	}

	events, err := h.repo.Timeline(c.UserContext(), id)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(response.Error{Error: err.Error()})
	}

	return c.JSON(response.AllocationTimeline{
		AllocationID: id,
		Data:         mapTimelineEvents(events),
		Count:        len(events),
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

func pathID(c *fiber.Ctx) (int64, bool) {
	id, err := strconv.ParseInt(c.Params("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

func queryCSV(value string, normalize func(string) string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, normalize(part))
	}
	return out
}

func mapAllocations(rows []repo.Allocation) []response.Allocation {
	out := make([]response.Allocation, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapAllocation(row))
	}
	return out
}

func mapAllocation(row repo.Allocation) response.Allocation {
	return response.Allocation{
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
	}
}

func mapTimelineEvents(rows []repo.AllocationTimelineEvent) []response.AllocationTimelineEvent {
	out := make([]response.AllocationTimelineEvent, 0, len(rows))
	for _, row := range rows {
		out = append(out, response.AllocationTimelineEvent{
			Time:        formatTime(row.EventTime),
			Stage:       row.Stage,
			Event:       row.Event,
			Status:      row.Status,
			Reason:      row.Reason,
			Source:      row.Source,
			ReferenceID: row.ReferenceID,
			Payload:     row.Payload,
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
