package repo

import (
	"time"

	"github.com/uptrace/bun"
)

const defaultTimeSeriesLimit = 100

type TimeSeriesFilter struct {
	Exchanges []string
	Bases     []string
	Quotes    []string
	From      time.Time
	To        time.Time
	Limit     int
}

func applyTimeSeriesFilters(q *bun.SelectQuery, alias string, filter TimeSeriesFilter) *bun.SelectQuery {
	col := func(name string) string {
		if alias == "" {
			return name
		}
		return alias + "." + name
	}

	if len(filter.Exchanges) > 0 {
		q = q.Where(col("exchange")+" IN (?)", bun.In(filter.Exchanges))
	}
	if len(filter.Bases) > 0 {
		q = q.Where(col("base")+" IN (?)", bun.In(filter.Bases))
	}
	if len(filter.Quotes) > 0 {
		q = q.Where(col("quote")+" IN (?)", bun.In(filter.Quotes))
	}
	if !filter.From.IsZero() {
		q = q.Where(col("time")+" >= ?", filter.From)
	}
	if !filter.To.IsZero() {
		q = q.Where(col("time")+" < ?", filter.To)
	}

	return q
}

func timeSeriesWhere(filter TimeSeriesFilter) ([]string, []any) {
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
	if !filter.From.IsZero() {
		where = append(where, "time >= ?")
		args = append(args, filter.From)
	}
	if !filter.To.IsZero() {
		where = append(where, "time < ?")
		args = append(args, filter.To)
	}

	return where, args
}

func normalizeTimeSeriesLimit(limit int) int {
	if limit <= 0 {
		return defaultTimeSeriesLimit
	}
	if limit > 500 {
		return 500
	}
	return limit
}
