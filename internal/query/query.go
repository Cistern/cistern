package query

import (
	"encoding/json"
	"time"
)

// Desc describes a query.
type Desc struct {
	Columns    []ColumnDesc `json:"columns,omitempty"`
	TimeRange  TimeRange    `json:"time_range"`
	GroupBy    []ColumnDesc `json:"group_by,omitempty"`
	Filters    []Filter     `json:"filters,omitempty"`
	PointSize  int64        `json:"point_size,omitempty"`
	OrderBy    []ColumnDesc `json:"order_by,omitempty"`
	Descending bool         `json:"descending"`
	Limit      int          `json:"limit,omitempty"`
}

// ColumnDesc describes a column.
type ColumnDesc struct {
	Name      string `json:"name"`
	Aggregate string `json:"aggregate,omitempty"`
}

// TimeRange represents start and end timestamps.
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// Filter represents a filter expression.
type Filter struct {
	Column    string      `json:"column"`
	Condition string      `json:"condition"`
	Value     interface{} `json:"value"`
}

func (d Desc) String() string {
	b, _ := json.Marshal(d)
	return string(b)
}
