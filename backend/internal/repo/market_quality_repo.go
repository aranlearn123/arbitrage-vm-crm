package repo

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/uptrace/bun"
)

const (
	defaultMarketQualityMinSamples        = 10
	defaultMarketQualityMaxSpreadBpsP50   = 10.0
	defaultMarketQualityMaxMidSpeedP95    = 50.0
	defaultMarketQualityMinDepthStability = 0.10
)

type MarketQualityRepo struct {
	db *bun.DB
}

type MarketQualityMetric struct {
	Time                 time.Time `bun:"time"`
	Exchange             string    `bun:"exchange"`
	Base                 string    `bun:"base"`
	Quote                string    `bun:"quote"`
	Samples              int       `bun:"samples"`
	SpreadBpsP50         string    `bun:"spread_bps_p50"`
	MidSpeedBpsPerSecP95 string    `bun:"mid_speed_bps_per_sec_p95"`
	DepthStabilityRatio  string    `bun:"depth_stability_ratio"`
}

type MarketQualityAlertFilter struct {
	TimeSeriesFilter
	MinSamples              int
	MaxSpreadBpsP50         float64
	MaxMidSpeedBpsPerSecP95 float64
	MinDepthStabilityRatio  float64
}

func NewMarketQualityRepo(db *bun.DB) *MarketQualityRepo {
	return &MarketQualityRepo{db: db}
}

func (r *MarketQualityRepo) Latest(ctx context.Context, filter TimeSeriesFilter) ([]MarketQualityMetric, error) {
	rows := make([]MarketQualityMetric, 0)
	where, args := timeSeriesWhere(filter)
	args = append(args, normalizeTimeSeriesLimit(filter.Limit))

	query := fmt.Sprintf(`
		WITH latest AS (
			SELECT DISTINCT ON (exchange, base, quote)
				time,
				exchange,
				base,
				quote,
				samples,
				spread_bps_p50,
				mid_speed_bps_per_sec_p95,
				depth_stability_ratio
			FROM market_quality_metrics_1m
			WHERE %s
			ORDER BY exchange ASC, base ASC, quote ASC, time DESC
		)
		SELECT
			time,
			exchange,
			base,
			quote,
			samples,
			spread_bps_p50::text AS spread_bps_p50,
			mid_speed_bps_per_sec_p95::text AS mid_speed_bps_per_sec_p95,
			depth_stability_ratio::text AS depth_stability_ratio
		FROM latest
		ORDER BY time DESC, exchange ASC, base ASC, quote ASC
		LIMIT ?
	`, strings.Join(where, " AND "))

	if err := r.db.NewRaw(query, args...).Scan(ctx, &rows); err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *MarketQualityRepo) History(ctx context.Context, filter TimeSeriesFilter) ([]MarketQualityMetric, error) {
	rows := make([]MarketQualityMetric, 0)
	q := r.baseMarketQualitySelect(&rows)
	q = applyTimeSeriesFilters(q, "mq", filter)

	if err := q.
		OrderExpr("mq.time DESC, mq.exchange ASC, mq.base ASC, mq.quote ASC").
		Limit(normalizeTimeSeriesLimit(filter.Limit)).
		Scan(ctx); err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *MarketQualityRepo) Alerts(ctx context.Context, filter MarketQualityAlertFilter) ([]MarketQualityMetric, error) {
	rows := make([]MarketQualityMetric, 0)
	filter = normalizeMarketQualityAlertFilter(filter)

	where, args := timeSeriesWhere(filter.TimeSeriesFilter)
	args = append(args,
		filter.MinSamples,
		filter.MaxSpreadBpsP50,
		filter.MaxMidSpeedBpsPerSecP95,
		filter.MinDepthStabilityRatio,
		normalizeTimeSeriesLimit(filter.Limit),
	)

	query := fmt.Sprintf(`
		WITH latest AS (
			SELECT DISTINCT ON (exchange, base, quote)
				time,
				exchange,
				base,
				quote,
				samples,
				spread_bps_p50,
				mid_speed_bps_per_sec_p95,
				depth_stability_ratio
			FROM market_quality_metrics_1m
			WHERE %s
			ORDER BY exchange ASC, base ASC, quote ASC, time DESC
		)
		SELECT
			time,
			exchange,
			base,
			quote,
			samples,
			spread_bps_p50::text AS spread_bps_p50,
			mid_speed_bps_per_sec_p95::text AS mid_speed_bps_per_sec_p95,
			depth_stability_ratio::text AS depth_stability_ratio
		FROM latest
		WHERE samples < ?
			OR spread_bps_p50 > ?
			OR mid_speed_bps_per_sec_p95 > ?
			OR depth_stability_ratio < ?
		ORDER BY time DESC, exchange ASC, base ASC, quote ASC
		LIMIT ?
	`, strings.Join(where, " AND "))

	if err := r.db.NewRaw(query, args...).Scan(ctx, &rows); err != nil {
		return nil, err
	}

	return rows, nil
}

func (r *MarketQualityRepo) baseMarketQualitySelect(model any) *bun.SelectQuery {
	return r.db.NewSelect().
		Model(model).
		ModelTableExpr("market_quality_metrics_1m AS mq").
		ColumnExpr("mq.time").
		ColumnExpr("mq.exchange").
		ColumnExpr("mq.base").
		ColumnExpr("mq.quote").
		ColumnExpr("mq.samples").
		ColumnExpr("mq.spread_bps_p50::text AS spread_bps_p50").
		ColumnExpr("mq.mid_speed_bps_per_sec_p95::text AS mid_speed_bps_per_sec_p95").
		ColumnExpr("mq.depth_stability_ratio::text AS depth_stability_ratio")
}

func normalizeMarketQualityAlertFilter(filter MarketQualityAlertFilter) MarketQualityAlertFilter {
	if filter.MinSamples <= 0 {
		filter.MinSamples = defaultMarketQualityMinSamples
	}
	if filter.MaxSpreadBpsP50 <= 0 {
		filter.MaxSpreadBpsP50 = defaultMarketQualityMaxSpreadBpsP50
	}
	if filter.MaxMidSpeedBpsPerSecP95 <= 0 {
		filter.MaxMidSpeedBpsPerSecP95 = defaultMarketQualityMaxMidSpeedP95
	}
	if filter.MinDepthStabilityRatio <= 0 {
		filter.MinDepthStabilityRatio = defaultMarketQualityMinDepthStability
	}
	return filter
}
