package main

import "time"

type QueryDesc struct {
	Columns    []ColumnDesc `json:"columns,omitempty"`
	TimeRange  TimeRange    `json:"time_range"`
	GroupBy    []string     `json:"group_by,omitempty"`
	Filters    []Filter     `json:"filters,omitempty"`
	PointSize  int64        `json:"point_size,omitempty"`
	OrderBy    []string     `json:"order_by,omitempty"`
	Descending bool         `json:"descending"`
	Limit      int          `json:"limit,omitempty"`
}

type ColumnDesc struct {
	Name      string `json:"name"`
	Aggregate string `json:"aggregate,omitempty"`
}

type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type Filter struct {
	Column    string      `json:"column"`
	Condition string      `json:"condition"`
	Value     interface{} `json:"value"`
}

type Event map[string]interface{}

type QueryResult struct {
	Summary []Event     `json:"summary,omitempty"`
	Series  []Event     `json:"series,omitempty"`
	Events  []Event     `json:"events,omitempty"`
	Query   interface{} `json:"query"`
}
