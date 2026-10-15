// Package history defines models and repository ports for long-term historical statistics.
package history

import (
	"context"
	"time"
)

// StatisticsRecord represents an aggregated metric rollup over a fixed time period.
type StatisticsRecord struct {
	ID          string    `json:"id"`
	EntityID    string    `json:"entity_id"`
	Period      string    `json:"period"` // "hourly", "daily"
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
	Mean        *float64  `json:"mean,omitempty"`
	Min         *float64  `json:"min,omitempty"`
	Max         *float64  `json:"max,omitempty"`
	Sum         *float64  `json:"sum,omitempty"`
	Count       int       `json:"count"`
	Unit        string    `json:"unit,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// StatisticsFilter specifies query parameters for long-term statistics retrieval.
type StatisticsFilter struct {
	EntityIDs []string  `json:"entity_ids"`
	Period    string    `json:"period"` // "hourly", "daily"
	From      time.Time `json:"from"`
	To        time.Time `json:"to"`
}

// StatisticsRepository handles persistence for pre-aggregated long-term metrics.
type StatisticsRepository interface {
	InsertOrUpdateBatch(ctx context.Context, stats []*StatisticsRecord) error
	GetStatistics(ctx context.Context, filter StatisticsFilter) ([]*StatisticsRecord, error)
}
