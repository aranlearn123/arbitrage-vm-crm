package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/uptrace/bun"
)

type FundingRepo struct {
	db *bun.DB
}

type Funding struct {
	Time        time.Time `bun:"time"`
	Exchange    string    `bun:"exchange"`
	Base        string    `bun:"base"`
	Quote       string    `bun:"quote"`
	FundingRate string    `bun:"funding_rate"`
}

type FundingSpread struct {
	Base         string    `bun:"base"`
	Quote        string    `bun:"quote"`
	BybitRate    string    `bun:"bybit_rate"`
	BitgetRate   string    `bun:"bitget_rate"`
	Spread       string    `bun:"spread"`
	SpreadBps    string    `bun:"spread_bps"`
	AbsSpreadBps string    `bun:"abs_spread_bps"`
	Direction    string    `bun:"direction_hint"`
	BybitTime    time.Time `bun:"bybit_time"`
	BitgetTime   time.Time `bun:"bitget_time"`
	Time         time.Time `bun:"time"`
}

type FundingSpreadFilter struct {
	Bases           []string
	Quotes          []string
	MinAbsSpreadBps float64
	Limit           int
}

func NewFundingRepo(db *bun.DB) *FundingRepo {
	return &FundingRepo{db: db}
}

func (r *FundingRepo) Latest(ctx context.Context, filter TimeSeriesFilter) ([]Funding, error) {
	rows := make([]Funding, 0)
	where, args := timeSeriesWhere(filter)
	args = append(args, normalizeTimeSeriesLimit(filter.Limit))

	query := fmt.Sprintf(`
		WITH latest AS (
			SELECT DISTINCT ON (exchange, base, quote)
				time,
				exchange,
				base,
				quote,
				funding_rate
			FROM funding
			WHERE %s
			ORDER BY exchange ASC, base ASC, quote ASC, time DESC
		)
		SELECT
			time,
			exchange,
			base,
			quote,
			funding_rate::text AS funding_rate
		FROM latest
		ORDER BY time DESC, exchange ASC, base ASC, quote ASC
		LIMIT ?
	`, strings.Join(where, " AND "))

	if err := r.db.NewRaw(query, args...).Scan(ctx, &rows); err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *FundingRepo) History(ctx context.Context, filter TimeSeriesFilter) ([]Funding, error) {
	rows := make([]Funding, 0)
	q := r.baseFundingSelect(&rows)
	q = applyTimeSeriesFilters(q, "f", filter)

	if err := q.
		OrderExpr("f.time DESC, f.exchange ASC, f.base ASC, f.quote ASC").
		Limit(normalizeTimeSeriesLimit(filter.Limit)).
		Scan(ctx); err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *FundingRepo) Spread(ctx context.Context, base string, quote string) (*FundingSpread, error) {
	row := new(FundingSpread)
	if err := r.db.NewRaw(`
		WITH latest AS (
			SELECT DISTINCT ON (exchange, base, quote)
				time,
				exchange,
				base,
				quote,
				funding_rate
			FROM funding
			WHERE exchange IN ('bybit', 'bitget')
				AND base = ?
				AND quote = ?
			ORDER BY exchange ASC, base ASC, quote ASC, time DESC
		),
		pivoted AS (
			SELECT
				base,
				quote,
				MAX(funding_rate) FILTER (WHERE exchange = 'bybit') AS bybit_rate,
				MAX(funding_rate) FILTER (WHERE exchange = 'bitget') AS bitget_rate,
				MAX(time) FILTER (WHERE exchange = 'bybit') AS bybit_time,
				MAX(time) FILTER (WHERE exchange = 'bitget') AS bitget_time
			FROM latest
			GROUP BY base, quote
		)
		SELECT
			base,
			quote,
			bybit_rate::text AS bybit_rate,
			bitget_rate::text AS bitget_rate,
			(bybit_rate - bitget_rate)::text AS spread,
			((bybit_rate - bitget_rate) * 10000)::text AS spread_bps,
			ABS((bybit_rate - bitget_rate) * 10000)::text AS abs_spread_bps,
			CASE
				WHEN bybit_rate > bitget_rate THEN 'short_bybit_long_bitget'
				WHEN bitget_rate > bybit_rate THEN 'short_bitget_long_bybit'
				ELSE 'flat'
			END AS direction_hint,
			bybit_time,
			bitget_time,
			GREATEST(bybit_time, bitget_time) AS time
		FROM pivoted
		WHERE bybit_rate IS NOT NULL
			AND bitget_rate IS NOT NULL
		LIMIT 1
	`, base, quote).Scan(ctx, row); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return row, nil
}

func (r *FundingRepo) TopSpreads(ctx context.Context, filter FundingSpreadFilter) ([]FundingSpread, error) {
	rows := make([]FundingSpread, 0)
	where, args := fundingSpreadWhere(filter)
	args = append(args, filter.MinAbsSpreadBps, normalizeTimeSeriesLimit(filter.Limit))

	query := fmt.Sprintf(`
		WITH latest AS (
			SELECT DISTINCT ON (exchange, base, quote)
				time,
				exchange,
				base,
				quote,
				funding_rate
			FROM funding
			WHERE %s
			ORDER BY exchange ASC, base ASC, quote ASC, time DESC
		),
		pivoted AS (
			SELECT
				base,
				quote,
				MAX(funding_rate) FILTER (WHERE exchange = 'bybit') AS bybit_rate,
				MAX(funding_rate) FILTER (WHERE exchange = 'bitget') AS bitget_rate,
				MAX(time) FILTER (WHERE exchange = 'bybit') AS bybit_time,
				MAX(time) FILTER (WHERE exchange = 'bitget') AS bitget_time
			FROM latest
			GROUP BY base, quote
		)
		SELECT
			base,
			quote,
			bybit_rate::text AS bybit_rate,
			bitget_rate::text AS bitget_rate,
			(bybit_rate - bitget_rate)::text AS spread,
			((bybit_rate - bitget_rate) * 10000)::text AS spread_bps,
			ABS((bybit_rate - bitget_rate) * 10000)::text AS abs_spread_bps,
			CASE
				WHEN bybit_rate > bitget_rate THEN 'short_bybit_long_bitget'
				WHEN bitget_rate > bybit_rate THEN 'short_bitget_long_bybit'
				ELSE 'flat'
			END AS direction_hint,
			bybit_time,
			bitget_time,
			GREATEST(bybit_time, bitget_time) AS time
		FROM pivoted
		WHERE bybit_rate IS NOT NULL
			AND bitget_rate IS NOT NULL
			AND ABS((bybit_rate - bitget_rate) * 10000) >= ?
		ORDER BY ABS((bybit_rate - bitget_rate) * 10000) DESC, base ASC, quote ASC
		LIMIT ?
	`, strings.Join(where, " AND "))

	if err := r.db.NewRaw(query, args...).Scan(ctx, &rows); err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *FundingRepo) baseFundingSelect(model any) *bun.SelectQuery {
	return r.db.NewSelect().
		Model(model).
		ModelTableExpr("funding AS f").
		ColumnExpr("f.time").
		ColumnExpr("f.exchange").
		ColumnExpr("f.base").
		ColumnExpr("f.quote").
		ColumnExpr("f.funding_rate::text AS funding_rate")
}

func fundingSpreadWhere(filter FundingSpreadFilter) ([]string, []any) {
	where := []string{"exchange IN ('bybit', 'bitget')"}
	args := make([]any, 0)

	if len(filter.Bases) > 0 {
		where = append(where, "base IN (?)")
		args = append(args, bun.In(filter.Bases))
	}
	if len(filter.Quotes) > 0 {
		where = append(where, "quote IN (?)")
		args = append(args, bun.In(filter.Quotes))
	}

	return where, args
}
