package repo

import (
	"context"
	"database/sql"
	"errors"
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

type AllocationListFilter struct {
	Statuses []string
	Bases    []string
	Quotes   []string
	Roles    []string
	Reasons  []string
	Limit    int
}

type AllocationTimelineEvent struct {
	EventTime   time.Time `bun:"event_time"`
	Stage       string    `bun:"stage"`
	Event       string    `bun:"event"`
	Status      string    `bun:"status"`
	Reason      string    `bun:"reason"`
	Source      string    `bun:"source"`
	ReferenceID string    `bun:"reference_id"`
	Payload     string    `bun:"payload"`
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

func (r *AllocationRepo) List(ctx context.Context, filter AllocationListFilter) ([]Allocation, error) {
	rows := make([]Allocation, 0)
	q := r.baseAllocationSelect(&rows)

	if len(filter.Statuses) > 0 {
		q = q.Where("a.status IN (?)", bun.In(filter.Statuses))
	}
	if len(filter.Bases) > 0 {
		q = q.Where("a.base IN (?)", bun.In(filter.Bases))
	}
	if len(filter.Quotes) > 0 {
		q = q.Where("a.quote IN (?)", bun.In(filter.Quotes))
	}
	if len(filter.Roles) > 0 {
		q = q.Where("a.role IN (?)", bun.In(filter.Roles))
	}
	if len(filter.Reasons) > 0 {
		q = q.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.WhereGroup(" OR ", func(q *bun.SelectQuery) *bun.SelectQuery {
				for _, reason := range filter.Reasons {
					q = q.WhereOr("a.note ILIKE ?", "%"+reason+"%")
				}
				return q
			})
		})
	}

	if err := q.
		OrderExpr("a.updated_at DESC, a.id DESC").
		Limit(normalizeLimit(filter.Limit)).
		Scan(ctx); err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *AllocationRepo) Cancelled(ctx context.Context, filter AllocationListFilter) ([]Allocation, error) {
	filter.Statuses = []string{"cancelled"}
	return r.List(ctx, filter)
}

func (r *AllocationRepo) GetByID(ctx context.Context, id int64) (*Allocation, error) {
	row := new(Allocation)
	if err := r.baseAllocationSelect(row).
		Where("a.id = ?", id).
		Limit(1).
		Scan(ctx); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return row, nil
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
	if err := r.baseAllocationSelect(&rows).
		Where("a.status IN (?)", bun.In(statuses)).
		OrderExpr("a.updated_at DESC, a.id DESC").
		Limit(limit).
		Scan(ctx); err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *AllocationRepo) Timeline(ctx context.Context, allocationID int64) ([]AllocationTimelineEvent, error) {
	rows := make([]AllocationTimelineEvent, 0)
	if err := r.db.NewRaw(`
		SELECT
			event_time,
			stage,
			event,
			status,
			reason,
			source,
			reference_id,
			payload
		FROM (
			SELECT
				a.created_at AS event_time,
				'allocation' AS stage,
				'created' AS event,
				a.status AS status,
				COALESCE(a.note, '') AS reason,
				'allocations' AS source,
				a.id::text AS reference_id,
				jsonb_build_object(
					'id', a.id,
					'base', a.base,
					'quote', a.quote,
					'status', a.status,
					'role', a.role,
					'budget_usd', a.budget_usd::text
				)::text AS payload
			FROM allocations AS a
			WHERE a.id = ?

			UNION ALL

			SELECT
				a.updated_at AS event_time,
				'allocation' AS stage,
				'status_' || a.status AS event,
				a.status AS status,
				COALESCE(a.note, '') AS reason,
				'allocations' AS source,
				a.id::text AS reference_id,
				jsonb_build_object(
					'id', a.id,
					'status', a.status,
					'note', COALESCE(a.note, ''),
					'worker_pid', COALESCE(a.worker_pid, '')
				)::text AS payload
			FROM allocations AS a
			WHERE a.id = ?
				AND a.updated_at <> a.created_at

			UNION ALL

			SELECT
				to_timestamp(sp.created_at::double precision / 1000000000.0) AS event_time,
				'sizing' AS stage,
				'scaling_plan_' || sp.status AS event,
				sp.status AS status,
				COALESCE(sp.payload->>'Reason', sp.payload->>'reason', '') AS reason,
				'scaling_plans' AS source,
				sp.plan_id AS reference_id,
				jsonb_build_object(
					'plan_id', sp.plan_id,
					'allocation_id', sp.allocation_id,
					'status', sp.status,
					'phase', sp.phase,
					'is_resumable', sp.is_resumable,
					'decision_seq', sp.decision_seq
				)::text AS payload
			FROM scaling_plans AS sp
			WHERE sp.allocation_id = ?
				AND sp.created_at > 0

			UNION ALL

			SELECT
				to_timestamp(rd.updated_at::double precision / 1000000000.0) AS event_time,
				'recovery' AS stage,
				'recovery_' || rd.action AS event,
				rd.action AS status,
				COALESCE(rd.payload->>'Reason', rd.payload->>'reason', '') AS reason,
				'recovery_decisions' AS source,
				rd.decision_seq::text AS reference_id,
				jsonb_build_object(
					'allocation_id', rd.allocation_id,
					'decision_seq', rd.decision_seq,
					'stage', rd.stage,
					'action', rd.action
				)::text AS payload
			FROM recovery_decisions AS rd
			WHERE rd.allocation_id = ?
				AND rd.updated_at > 0

			UNION ALL

			SELECT
				to_timestamp(om.updated_at::double precision / 1000000000.0) AS event_time,
				'order_routing' AS stage,
				CASE
					WHEN om.cancel_requested THEN 'cancel_requested'
					ELSE 'routing_state'
				END AS event,
				om.last_terminal_status AS status,
				COALESCE(om.cancel_reason, '') AS reason,
				'order_management_routing_states' AS source,
				om.plan_id AS reference_id,
				jsonb_build_object(
					'plan_id', om.plan_id,
					'allocation_id', om.allocation_id,
					'slice_index', om.slice_index,
					'lead_exchange', om.lead_exchange,
					'follow_exchange', om.follow_exchange,
					'direction', om.direction,
					'exec_mode', om.exec_mode,
					'requested_notional', om.requested_notional::text
				)::text AS payload
			FROM order_management_routing_states AS om
			WHERE om.allocation_id = ?
				AND om.updated_at > 0

			UNION ALL

			SELECT
				to_timestamp(op.occurred_at::double precision / 1000000000.0) AS event_time,
				'order_progress' AS stage,
				'progress_' || op.status AS event,
				op.status AS status,
				COALESCE(op.reason, '') AS reason,
				'order_management_progress_events' AS source,
				op.plan_id || ':' || op.progress_seq::text AS reference_id,
				jsonb_build_object(
					'plan_id', op.plan_id,
					'progress_seq', op.progress_seq,
					'allocation_id', op.allocation_id,
					'slice_index', op.slice_index,
					'submitted_notional', op.submitted_notional::text,
					'funding_filled_delta_notional', op.funding_filled_delta_notional::text,
					'hedge_filled_delta_notional', op.hedge_filled_delta_notional::text
				)::text AS payload
			FROM order_management_progress_events AS op
			WHERE op.allocation_id = ?
				AND op.occurred_at > 0
		) AS events
		ORDER BY event_time ASC, source ASC, reference_id ASC
	`, allocationID, allocationID, allocationID, allocationID, allocationID, allocationID).Scan(ctx, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}

func (r *AllocationRepo) baseAllocationSelect(model any) *bun.SelectQuery {
	return r.db.NewSelect().
		Model(model).
		ModelTableExpr("allocations AS a").
		ColumnExpr("a.id").
		ColumnExpr("a.base").
		ColumnExpr("a.quote").
		ColumnExpr("a.direction").
		ColumnExpr("a.rank").
		ColumnExpr("a.score::text AS score").
		ColumnExpr("a.role").
		ColumnExpr("a.status").
		ColumnExpr("a.budget_usd::text AS budget_usd").
		ColumnExpr("a.worker_pid").
		ColumnExpr("a.note").
		ColumnExpr("a.created_at").
		ColumnExpr("a.updated_at")
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
