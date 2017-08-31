package query

import (
	"encoding/json"
	"strconv"
	"time"
)

type expression struct {
	query          Desc
	currentSection string
}

func (e *expression) AddColumn() {
	switch e.currentSection {
	case "columns":
		e.query.Columns = append(e.query.Columns, ColumnDesc{})
	case "group by":
		e.query.GroupBy = append(e.query.GroupBy, ColumnDesc{})
	case "order by":
		e.query.OrderBy = append(e.query.OrderBy, ColumnDesc{})
	}
}

func (e *expression) SetColumnName(name string) {
	switch e.currentSection {
	case "columns":
		e.query.Columns[len(e.query.Columns)-1].Name = name
	case "group by":
		e.query.GroupBy[len(e.query.GroupBy)-1].Name = name
	case "order by":
		e.query.OrderBy[len(e.query.OrderBy)-1].Name = name
	}
}

func (e *expression) SetColumnAggregate(aggregate string) {
	switch e.currentSection {
	case "columns":
		e.query.Columns[len(e.query.Columns)-1].Aggregate = aggregate
	case "group by":
		e.query.GroupBy[len(e.query.GroupBy)-1].Aggregate = aggregate
	case "order by":
		e.query.OrderBy[len(e.query.OrderBy)-1].Aggregate = aggregate
	}
}

func (e *expression) AddFilter() {
	e.query.Filters = append(e.query.Filters, Filter{})
}

func (e *expression) SetFilterColumn(column string) {
	e.query.Filters[len(e.query.Filters)-1].Column = column
}

func (e *expression) SetFilterCondition(condition string) {
	e.query.Filters[len(e.query.Filters)-1].Condition = condition
}

func (e *expression) SetFilterValue(value string) {
	e.query.Filters[len(e.query.Filters)-1].Value = json.RawMessage(value)
}

func (e *expression) SetDescending() {
	e.query.Descending = true
}

func (e *expression) SetLimit(num string) {
	e.query.Limit, _ = strconv.Atoi(num)
}

func (e *expression) SetPointSize(num string) {
	dur, _ := time.ParseDuration(num)
	e.query.PointSize = int64(dur) / 1e3
}

func Parse(query string) (*Desc, error) {
	p := &parser{
		Buffer: query,
	}
	p.Init()
	err := p.Parse()
	if err != nil {
		return nil, err
	}
	p.Execute()
	return &p.query, nil
}
