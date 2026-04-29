package repo

import (
	"context"
	"time"

	"github.com/uptrace/bun"
)

const defaultAllocationLimit = 100

var activeAllocationStatuses = []string{"created", "running", "failed", "paused"}

type AllocationRepo struct {
	db *bun.DB
}

type Allocation struct {
	ID        int64     `bun:"id"`
	Base      string    `bun:"base"`
	Quote     string    `bun:"quote"`
	Direction string    `bun:"direction"`
	Rank      int       `bun:"rank"`
	Score     string    `bun:"score"`
	Role      string    `bun:"role"`
	Status    string    `bun:"status"`
	BudgetUSD string    `bun:"budget_usd"`
	WorkerPID *string   `bun:"worker_pid"`
	Note      *string   `bun:"note"`
	CreatedAt time.Time `bun:"created_at"`
	UpdatedAt time.Time `bun:"updated_at"`
}

type AllocationStatusCount struct {
	Status string `bun:"status"`
	Count  int64  `bun:"count"`
}

type CancelledReasonCount struct {
	Reason string `bun:"reason"`
	Count  int64  `bun:"count"`
}

type AllocationSummary struct {
	Total           int64
	Active          int64 `bun:"active"`
	Cancelled       int64
	ActiveBudgetUSD string `bun:"active_budget_usd"`
	ByStatus        []AllocationStatusCount
}

func NewAllocationRepo(db *bun.DB) *AllocationRepo {
	return &AllocationRepo{db: db}
}

func (r *AllocationRepo) Summary(ctx context.Context) (AllocationSummary, error) {
	var summary AllocationSummary

	if err := r.db.NewRaw(`
		SELECT status, COUNT(*) AS count
		FROM allocations
		GROUP BY status
		ORDER BY status
	`).Scan(ctx, &summary.ByStatus); err != nil {
		return AllocationSummary{}, err
	}

	for _, row := range summary.ByStatus {
		summary.Total += row.Count
		if row.Status == "cancelled" {
			summary.Cancelled = row.Count
		}
	}

	if err := r.db.NewRaw(`
		SELECT
			COUNT(*) AS active,
			COALESCE(SUM(budget_usd), 0)::text AS active_budget_usd
		FROM allocations
		WHERE status IN (?)
	`, bun.In(activeAllocationStatuses)).Scan(ctx, &summary); err != nil {
		return AllocationSummary{}, err
	}

	return summary, nil
}

func (r *AllocationRepo) Active(ctx context.Context, limit int) ([]Allocation, error) {
	return r.listByStatuses(ctx, activeAllocationStatuses, normalizeLimit(limit))
}

func (r *AllocationRepo) Running(ctx context.Context, limit int) ([]Allocation, error) {
	return r.listByStatuses(ctx, []string{"running"}, normalizeLimit(limit))
}

func (r *AllocationRepo) CancelledReasons(ctx context.Context, limit int) ([]CancelledReasonCount, error) {
	rows := make([]CancelledReasonCount, 0)
	if err := r.db.NewRaw(`
		WITH normalized AS (
			SELECT
				COALESCE(
					NULLIF(TRIM(SPLIT_PART(note, ';', 1)), ''),
					'unknown'
				) AS reason
			FROM allocations
			WHERE status = 'cancelled'
		)
		SELECT reason, COUNT(*) AS count
		FROM normalized
		GROUP BY reason
		ORDER BY count DESC, reason ASC
		LIMIT ?
	`, normalizeLimit(limit)).Scan(ctx, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *AllocationRepo) listByStatuses(ctx context.Context, statuses []string, limit int) ([]Allocation, error) {
	rows := make([]Allocation, 0)
	if err := r.db.NewRaw(`
		SELECT
			id,
			base,
			quote,
			direction,
			rank,
			score::text AS score,
			role,
			status,
			budget_usd::text AS budget_usd,
			worker_pid,
			note,
			created_at,
			updated_at
		FROM allocations
		WHERE status IN (?)
		ORDER BY updated_at DESC, id DESC
		LIMIT ?
	`, bun.In(statuses), limit).Scan(ctx, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return defaultAllocationLimit
	}
	if limit > 500 {
		return 500
	}
	return limit
}
