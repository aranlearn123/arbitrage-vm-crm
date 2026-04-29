package repo

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"
)

type SystemRepo struct {
	db *bun.DB
}

type SystemOverview struct {
	TimescaleEnabled       bool         `bun:"timescale_enabled"`
	FundingLastAt          sql.NullTime `bun:"funding_last_at"`
	OpenInterestLastAt     sql.NullTime `bun:"open_interest_last_at"`
	MarketQualityLastAt    sql.NullTime `bun:"market_quality_last_at"`
	AllocationLastUpdated  sql.NullTime `bun:"allocation_last_updated_at"`
	AllocationCount        int64        `bun:"allocation_count"`
	RunningAllocationCount int64        `bun:"running_allocation_count"`
}

type SystemExchangeOverview struct {
	Exchange            string       `bun:"exchange"`
	LastFundingAt       sql.NullTime `bun:"last_funding_at"`
	LastOpenInterestAt  sql.NullTime `bun:"last_open_interest_at"`
	LastMarketQualityAt sql.NullTime `bun:"last_market_quality_at"`
}

func NewSystemRepo(db *bun.DB) *SystemRepo {
	return &SystemRepo{db: db}
}

func (r *SystemRepo) Overview(ctx context.Context) (SystemOverview, error) {
	row := SystemOverview{}
	if err := r.db.NewRaw(`
		SELECT
			EXISTS (
				SELECT 1
				FROM pg_extension
				WHERE extname = 'timescaledb'
			) AS timescale_enabled,
			(SELECT MAX(time) FROM funding) AS funding_last_at,
			(SELECT MAX(time) FROM open_interest) AS open_interest_last_at,
			(SELECT MAX(time) FROM market_quality_metrics_1m) AS market_quality_last_at,
			(SELECT MAX(updated_at) FROM allocations) AS allocation_last_updated_at,
			(SELECT COUNT(*) FROM allocations) AS allocation_count,
			(SELECT COUNT(*) FROM allocations WHERE status = 'running') AS running_allocation_count
	`).Scan(ctx, &row); err != nil {
		return SystemOverview{}, err
	}
	return row, nil
}

func (r *SystemRepo) Exchanges(ctx context.Context) ([]SystemExchangeOverview, error) {
	rows := make([]SystemExchangeOverview, 0)
	if err := r.db.NewRaw(`
		WITH names AS (
			SELECT exchange FROM funding
			UNION
			SELECT exchange FROM open_interest
			UNION
			SELECT exchange FROM market_quality_metrics_1m
		),
		funding_latest AS (
			SELECT exchange, MAX(time) AS last_funding_at
			FROM funding
			GROUP BY exchange
		),
		open_interest_latest AS (
			SELECT exchange, MAX(time) AS last_open_interest_at
			FROM open_interest
			GROUP BY exchange
		),
		market_quality_latest AS (
			SELECT exchange, MAX(time) AS last_market_quality_at
			FROM market_quality_metrics_1m
			GROUP BY exchange
		)
		SELECT
			n.exchange,
			f.last_funding_at,
			oi.last_open_interest_at,
			mq.last_market_quality_at
		FROM names AS n
		LEFT JOIN funding_latest AS f ON f.exchange = n.exchange
		LEFT JOIN open_interest_latest AS oi ON oi.exchange = n.exchange
		LEFT JOIN market_quality_latest AS mq ON mq.exchange = n.exchange
		ORDER BY n.exchange ASC
	`).Scan(ctx, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}
