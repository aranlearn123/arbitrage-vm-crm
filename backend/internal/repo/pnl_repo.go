package repo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/uptrace/bun"
)

type PnLRepo struct {
	db *bun.DB
}

type PnLEvent struct {
	EventTime  time.Time `bun:"event_time"`
	Exchange   string    `bun:"exchange"`
	Base       string    `bun:"base"`
	Quote      string    `bun:"quote"`
	Component  string    `bun:"component"`
	Amount     string    `bun:"amount"`
	Currency   string    `bun:"currency"`
	SourceType string    `bun:"source_type"`
	SourceID   string    `bun:"source_id"`
	CreatedAt  time.Time `bun:"created_at"`
}

type PnLFilter struct {
	Exchanges   []string
	Bases       []string
	Quotes      []string
	Components  []string
	SourceTypes []string
	SourceIDs   []string
	From        time.Time
	To          time.Time
	Limit       int
}

type PnLSummary struct {
	Count            int64  `bun:"count"`
	TotalAmount      string `bun:"total_amount"`
	FundingAmount    string `bun:"funding_amount"`
	TradingFeeAmount string `bun:"trading_fee_amount"`
	TradingPnLAmount string `bun:"trading_pnl_amount"`
}

type PnLGroup struct {
	Exchange         string    `bun:"exchange"`
	Base             string    `bun:"base"`
	Quote            string    `bun:"quote"`
	Component        string    `bun:"component"`
	Count            int64     `bun:"count"`
	TotalAmount      string    `bun:"total_amount"`
	FundingAmount    string    `bun:"funding_amount"`
	TradingFeeAmount string    `bun:"trading_fee_amount"`
	TradingPnLAmount string    `bun:"trading_pnl_amount"`
	FirstEventTime   time.Time `bun:"first_event_time"`
	LastEventTime    time.Time `bun:"last_event_time"`
}

func NewPnLRepo(db *bun.DB) *PnLRepo {
	return &PnLRepo{db: db}
}

func (r *PnLRepo) Events(ctx context.Context, filter PnLFilter) ([]PnLEvent, error) {
	rows := make([]PnLEvent, 0)
	q := r.db.NewSelect().
		Model(&rows).
		ModelTableExpr("pnl_events AS p").
		ColumnExpr("p.event_time").
		ColumnExpr("p.exchange").
		ColumnExpr("p.base").
		ColumnExpr("p.quote").
		ColumnExpr("p.component").
		ColumnExpr("p.amount::text AS amount").
		ColumnExpr("p.currency").
		ColumnExpr("p.source_type").
		ColumnExpr("p.source_id").
		ColumnExpr("p.created_at")

	q = applyPnLFilters(q, filter)

	if err := q.
		OrderExpr("p.event_time DESC, p.exchange ASC, p.base ASC, p.quote ASC, p.component ASC").
		Limit(normalizeTimeSeriesLimit(filter.Limit)).
		Scan(ctx); err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *PnLRepo) Summary(ctx context.Context, filter PnLFilter) (PnLSummary, error) {
	row := PnLSummary{}
	where, args := pnlWhere(filter)

	query := fmt.Sprintf(`
		SELECT
			COUNT(*) AS count,
			COALESCE(SUM(amount), 0)::text AS total_amount,
			COALESCE(SUM(amount) FILTER (WHERE component = 'funding'), 0)::text AS funding_amount,
			COALESCE(SUM(amount) FILTER (WHERE component = 'trading_fee'), 0)::text AS trading_fee_amount,
			COALESCE(SUM(amount) FILTER (WHERE component = 'trading_pnl'), 0)::text AS trading_pnl_amount
		FROM pnl_events
		WHERE %s
	`, strings.Join(where, " AND "))

	if err := r.db.NewRaw(query, args...).Scan(ctx, &row); err != nil {
		return PnLSummary{}, err
	}

	return row, nil
}

func (r *PnLRepo) ByPair(ctx context.Context, filter PnLFilter) ([]PnLGroup, error) {
	return r.groups(ctx, filter, []string{"base", "quote"}, "base ASC, quote ASC")
}

func (r *PnLRepo) ByExchange(ctx context.Context, filter PnLFilter) ([]PnLGroup, error) {
	return r.groups(ctx, filter, []string{"exchange"}, "exchange ASC")
}

func (r *PnLRepo) ByComponent(ctx context.Context, filter PnLFilter) ([]PnLGroup, error) {
	return r.groups(ctx, filter, []string{"component"}, "component ASC")
}

func (r *PnLRepo) groups(ctx context.Context, filter PnLFilter, groupCols []string, orderExpr string) ([]PnLGroup, error) {
	rows := make([]PnLGroup, 0)
	where, args := pnlWhere(filter)
	args = append(args, normalizeTimeSeriesLimit(filter.Limit))

	selectCols := make([]string, 0, len(groupCols))
	for _, col := range groupCols {
		selectCols = append(selectCols, col)
	}

	query := fmt.Sprintf(`
		SELECT
			%s,
			COUNT(*) AS count,
			COALESCE(SUM(amount), 0)::text AS total_amount,
			COALESCE(SUM(amount) FILTER (WHERE component = 'funding'), 0)::text AS funding_amount,
			COALESCE(SUM(amount) FILTER (WHERE component = 'trading_fee'), 0)::text AS trading_fee_amount,
			COALESCE(SUM(amount) FILTER (WHERE component = 'trading_pnl'), 0)::text AS trading_pnl_amount,
			MIN(event_time) AS first_event_time,
			MAX(event_time) AS last_event_time
		FROM pnl_events
		WHERE %s
		GROUP BY %s
		ORDER BY %s
		LIMIT ?
	`, strings.Join(selectCols, ", "), strings.Join(where, " AND "), strings.Join(groupCols, ", "), orderExpr)

	if err := r.db.NewRaw(query, args...).Scan(ctx, &rows); err != nil {
		return nil, err
	}

	return rows, nil
}

func applyPnLFilters(q *bun.SelectQuery, filter PnLFilter) *bun.SelectQuery {
	if len(filter.Exchanges) > 0 {
		q = q.Where("p.exchange IN (?)", bun.In(filter.Exchanges))
	}
	if len(filter.Bases) > 0 {
		q = q.Where("p.base IN (?)", bun.In(filter.Bases))
	}
	if len(filter.Quotes) > 0 {
		q = q.Where("p.quote IN (?)", bun.In(filter.Quotes))
	}
	if len(filter.Components) > 0 {
		q = q.Where("p.component IN (?)", bun.In(filter.Components))
	}
	if len(filter.SourceTypes) > 0 {
		q = q.Where("p.source_type IN (?)", bun.In(filter.SourceTypes))
	}
	if len(filter.SourceIDs) > 0 {
		q = q.Where("p.source_id IN (?)", bun.In(filter.SourceIDs))
	}
	if !filter.From.IsZero() {
		q = q.Where("p.event_time >= ?", filter.From)
	}
	if !filter.To.IsZero() {
		q = q.Where("p.event_time < ?", filter.To)
	}
	return q
}

func pnlWhere(filter PnLFilter) ([]string, []any) {
	where := []string{"TRUE"}
	args := make([]any, 0)

	if len(filter.Exchanges) > 0 {
		where = append(where, "exchange IN (?)")
		args = append(args, bun.In(filter.Exchanges))
	}
	if len(filter.Bases) > 0 {
		where = append(where, "base IN (?)")
		args = append(args, bun.In(filter.Bases))
	}
	if len(filter.Quotes) > 0 {
		where = append(where, "quote IN (?)")
		args = append(args, bun.In(filter.Quotes))
	}
	if len(filter.Components) > 0 {
		where = append(where, "component IN (?)")
		args = append(args, bun.In(filter.Components))
	}
	if len(filter.SourceTypes) > 0 {
		where = append(where, "source_type IN (?)")
		args = append(args, bun.In(filter.SourceTypes))
	}
	if len(filter.SourceIDs) > 0 {
		where = append(where, "source_id IN (?)")
		args = append(args, bun.In(filter.SourceIDs))
	}
	if !filter.From.IsZero() {
		where = append(where, "event_time >= ?")
		args = append(args, filter.From)
	}
	if !filter.To.IsZero() {
		where = append(where, "event_time < ?")
		args = append(args, filter.To)
	}

	return where, args
}
