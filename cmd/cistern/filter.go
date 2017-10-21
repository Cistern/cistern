package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"github.com/Cistern/cistern/internal/query"
)

type FilterType int

const (
	FilterUnknown FilterType = iota
	FilterEquals
	FilterNotEquals
	FilterLessThan
	FilterLessThanOrEqual
	FilterGreaterThan
	FilterGreaterThanOrEqual
	FilterMatches
)

func (f FilterType) String() string {
	rep := map[FilterType]string{
		FilterEquals:             "=",
		FilterNotEquals:          "!=",
		FilterLessThan:           "<",
		FilterLessThanOrEqual:    "<=",
		FilterGreaterThan:        ">",
		FilterGreaterThanOrEqual: ">=",
		FilterMatches:            "matches",
	}
	if str, ok := rep[f]; ok {
		return str
	}
	return "?"
}

func stringToFilterType(s string) FilterType {
	ft := FilterUnknown
	rep := map[string]FilterType{
		"=":       FilterEquals,
		"!=":      FilterNotEquals,
		"<":       FilterLessThan,
		"<=":      FilterLessThanOrEqual,
		">":       FilterGreaterThan,
		">=":      FilterGreaterThanOrEqual,
		"matches": FilterMatches,
	}
	if f, ok := rep[s]; ok {
		ft = f
	}
	return ft
}

func buildFilters(queryFilters []query.Filter) ([]Filter, error) {
	filters := []Filter{}

	for _, f := range queryFilters {
		filterType := stringToFilterType(f.Condition)
		switch filterType {
		case FilterUnknown:
			return nil, fmt.Errorf("unknown filter %s", f.Condition)

		case FilterEquals:
			filters = append(filters, EqualsFilter(f.Column, f.Value))
		case FilterNotEquals:
			filters = append(filters, NotEqualsFilter(f.Column, f.Value))
		case FilterLessThan:
			filters = append(filters, LessThanFilter(f.Column, f.Value))
		case FilterLessThanOrEqual:
			filters = append(filters, LessThanOrEqualFilter(f.Column, f.Value))
		case FilterGreaterThan:
			filters = append(filters, GreaterThanFilter(f.Column, f.Value))
		case FilterGreaterThanOrEqual:
			filters = append(filters, GreaterThanOrEqualFilter(f.Column, f.Value))
		case FilterMatches:
			str, ok := f.Value.(string)
			if !ok {
				return nil, fmt.Errorf("expected string value for matches filter")
			}
			r, err := regexp.Compile(str)
			if err != nil {
				return nil, err
			}
			filters = append(filters, MatchesFilter(f.Column, r))
		}
	}

	return filters, nil
}

type Filter struct {
	column     string
	value      interface{}
	filterFunc func(a, b interface{}) bool
}

func (f Filter) Filter(e Event) bool {
	if v, ok := e[f.column]; !ok {
		return false
	} else {
		return f.filterFunc(v, f.value)
	}
}

func EqualsFilter(column string, value interface{}) Filter {
	filterFunc := func(a, b interface{}) bool {
		return compareInterfaces(a, b) == 0
	}
	return Filter{
		column:     column,
		value:      value,
		filterFunc: filterFunc,
	}
}

func NotEqualsFilter(column string, value interface{}) Filter {
	filterFunc := func(a, b interface{}) bool {
		return compareInterfaces(a, b) != 0
	}
	return Filter{
		column:     column,
		value:      value,
		filterFunc: filterFunc,
	}
}

func LessThanFilter(column string, value interface{}) Filter {
	filterFunc := func(a, b interface{}) bool {
		return compareInterfaces(a, b) < 0
	}
	return Filter{
		column:     column,
		value:      value,
		filterFunc: filterFunc,
	}
}

func LessThanOrEqualFilter(column string, value interface{}) Filter {
	filterFunc := func(a, b interface{}) bool {
		return compareInterfaces(a, b) <= 0
	}
	return Filter{
		column:     column,
		value:      value,
		filterFunc: filterFunc,
	}
}

func GreaterThanFilter(column string, value interface{}) Filter {
	filterFunc := func(a, b interface{}) bool {
		return compareInterfaces(a, b) > 0
	}
	return Filter{
		column:     column,
		value:      value,
		filterFunc: filterFunc,
	}
}

func GreaterThanOrEqualFilter(column string, value interface{}) Filter {
	filterFunc := func(a, b interface{}) bool {
		return compareInterfaces(a, b) >= 0
	}
	return Filter{
		column:     column,
		value:      value,
		filterFunc: filterFunc,
	}
}

func MatchesFilter(column string, r *regexp.Regexp) Filter {
	filterFunc := func(a, b interface{}) bool {
		aString, ok := a.(string)
		if !ok {
			return false
		}
		return r.MatchString(aString)
	}
	return Filter{
		column:     column,
		filterFunc: filterFunc,
	}
}

func checkEquals(a, b interface{}) bool {
	return compareInterfaces(a, b) == 0
}

func compareInterfaces(a, b interface{}) int {
	switch a.(type) {
	case int:
		aInt := a.(int)
		if bInt, ok := b.(int); ok {
			return aInt - bInt
		}
	case float64:
		aFloat := a.(float64)
		if bFloat, ok := b.(float64); ok {
			if aFloat == bFloat {
				return 0
			} else if aFloat < bFloat {
				return -1
			} else {
				return 1
			}
		}
	case string:
		aString := a.(string)
		if bString, ok := b.(string); ok {
			if aString == bString {
				return 0
			} else if aString < bString {
				return -1
			} else {
				return 1
			}
		}
	case json.Number:
		aFloat, _ := strconv.ParseFloat(string(a.(json.Number)), 64)
		if bNumber, ok := b.(json.Number); ok {
			bFloat, _ := strconv.ParseFloat(string(bNumber), 64)
			if aFloat == bFloat {
				return 0
			} else if aFloat < bFloat {
				return -1
			} else {
				return 1
			}
		}
	}
	return -1
}
