package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	commaSeparatedGroups = regexp.MustCompile(`(?:"(?:\\.|[^"])*"|\\.|[^,])+`)
)

func parseQuery(columnsStr, groupStr, filters, orderBy string) (QueryDesc, error) {
	result := QueryDesc{}

	if columnsStr != "" {
		columnParts := strings.Split(columnsStr, " ")
		for _, column := range columnParts {
			parts := strings.Split(column, "(")
			if len(parts) != 2 {
				return result, errors.New("bad columns")
			}
			result.Columns = append(result.Columns, ColumnDesc{
				Aggregate: parts[0],
				Name:      strings.TrimRight(parts[1], "),"),
			})
		}
	}

	if groupStr != "" {
		parts := strings.Split(groupStr, ",")
		for _, groupColumn := range parts {
			result.GroupBy = append(result.GroupBy, strings.TrimSpace(groupColumn))
		}
	}

	if filters != "" {
		filterParts := commaSeparatedGroups.FindAllString(filters, -1)
		for _, filterPart := range filterParts {

			splitFilterParts := strings.SplitN(strings.TrimSpace(filterPart), " ", 3)
			jsonValue := json.RawMessage(splitFilterParts[2])
			_, err := json.Marshal(jsonValue)
			if err != nil {
				return result, fmt.Errorf("%s is not a valid JSON value. Are you missing quotes?", jsonValue)
			}
			result.Filters = append(result.Filters, Filter{
				Column:    splitFilterParts[0],
				Condition: splitFilterParts[1],
				Value:     jsonValue,
			})
		}
	}

	if orderBy != "" {
		parts := strings.Split(orderBy, ",")
		for _, orderColumn := range parts {
			result.OrderBy = append(result.OrderBy, strings.TrimSpace(orderColumn))
		}
	}

	return result, nil
}
