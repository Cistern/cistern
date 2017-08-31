package main

import (
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Cistern/cistern/internal/query"
)

type QueryResult struct {
	Summary []Event     `json:"summary,omitempty"`
	Series  []Event     `json:"series,omitempty"`
	Events  []Event     `json:"events,omitempty"`
	Query   interface{} `json:"query"`
}

type ByTimestamp []Event

func (t ByTimestamp) Len() int      { return len(t) }
func (t ByTimestamp) Swap(i, j int) { t[i], t[j] = t[j], t[i] }
func (t ByTimestamp) Less(i, j int) bool {
	return t[i]["_ts"].(time.Time).Before(t[j]["_ts"].(time.Time))
}

type OrderBy struct {
	columns []string
	events  []Event
}

func (o OrderBy) Len() int      { return len(o.events) }
func (o OrderBy) Swap(i, j int) { o.events[i], o.events[j] = o.events[j], o.events[i] }
func (o OrderBy) Less(i, j int) bool {
	for _, col := range o.columns {
		if compareInterfaces(o.events[i][col], o.events[j][col]) >= 0 {
			return false
		}
	}
	return true
}

func (c *EventCollection) Query(desc query.Desc) (*QueryResult, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	if desc.TimeRange.Start.Before(minTimestamp) {
		desc.TimeRange.Start = minTimestamp
	}
	if desc.TimeRange.End.Before(minTimestamp) {
		desc.TimeRange.End = minTimestamp
	}

	if desc.TimeRange.Start == minTimestamp && desc.TimeRange.End == minTimestamp {
		desc.TimeRange.End = fromMicrosecondTime(math.MaxInt64)
	}

	cur, err := c.col.NewCursor()
	if err != nil {
		return nil, err
	}

	formattedStartTs := formatTs(toMicrosecondTime(desc.TimeRange.Start))
	formattedEndTs := formatTs(toMicrosecondTime(desc.TimeRange.End))

	startKey := string(eventKeyPrefix) + string(formattedStartTs[:])
	endKey := string(eventKeyPrefix) + string(formattedEndTs[:]) + "\xff"

	summaryRows := map[string][]float64{}
	summaryRowsByTime := map[int64]map[string][]float64{}
	resultEvents := []Event{}

	cur.Seek(startKey)

CursorLoop:
	for cur.Next() {
		if cur.Key() > endKey {
			break
		}

		if (cur.Key())[0] == '_' {
			continue
		}

		// Extract event
		id := cur.Key()
		val := cur.Value()
		ts, keyTag, hash, err := splitCollectionID(id)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		if ts < toMicrosecondTime(desc.TimeRange.Start) {
			continue CursorLoop
		}

		event := Event{}
		valBytes := []byte(val)
		err = json.Unmarshal(valBytes, &event)
		if err != nil {
			log.Println(err)
			return nil, err
		}

		eventID := strconv.FormatInt(ts, 10) + "|" + keyTag
		event["_ts"] = ts
		event["_tag"] = keyTag
		if len(hash) > 0 {
			event["_hash"] = hash
			eventID += "|" + hash
		}
		event["_id"] = eventID

		// Apply filters
		for _, filter := range desc.Filters {
			if colValue, ok := event[filter.Column]; ok {
				filterResult := false
				switch filter.Condition {
				case "eq":
					filterResult = checkEquals(colValue, filter.Value)
				case "neq":
					filterResult = !checkEquals(colValue, filter.Value)
				default:
					return nil, errors.New("invalid filter condition")
				}

				if !filterResult {
					continue CursorLoop
				}
			} else {
				continue CursorLoop
			}
		}

		if len(desc.GroupBy) == 0 && len(desc.Columns) == 0 && desc.PointSize <= 0 {
			// No group by or aggregates
			event["_ts"] = fromMicrosecondTime(ts)
			resultEvents = append(resultEvents, event)
			if desc.Limit > 0 && len(resultEvents) == desc.Limit {
				break
			}
			continue
		}

		// Figure out the row key for grouping
		rowKey := ""
		if len(desc.GroupBy) > 0 {
			rowKeyParts := []string{}
			for _, groupCol := range desc.GroupBy {
				groupColVal := event[groupCol.Name] // TODO: support aggregates on grouped columns
				if groupColVal == nil {
					continue CursorLoop
				}
				marshaledColVal, err := json.Marshal(groupColVal)
				if err != nil {
					continue CursorLoop
				}
				rowKeyParts = append(rowKeyParts, string(marshaledColVal))
			}
			rowKey = strings.Join(rowKeyParts, "\x00")
		}

		// Do the aggregations.

		updateRows := func(rowKey string, rows map[string][]float64) {
			rowAggregates, ok := rows[rowKey]
			if !ok {
				rowAggregates = make([]float64, len(desc.Columns))
				for i := range rowAggregates {
					rowAggregates[i] = math.NaN()
				}
			}

			for i, columnDesc := range desc.Columns {
				floatVal := 0.0
				columnVal := event[columnDesc.Name]
				switch columnVal.(type) {
				case int:
					floatVal = float64(columnVal.(int))
				case float64:
					floatVal = columnVal.(float64)
				}
				switch columnDesc.Aggregate {
				case "sum":
					if math.IsNaN(rowAggregates[i]) {
						rowAggregates[i] = 0
					}
					rowAggregates[i] += floatVal
				case "count":
					if math.IsNaN(rowAggregates[i]) {
						rowAggregates[i] = 0
					}
					rowAggregates[i] += 1
				case "min":
					if rowAggregates[i] > floatVal || math.IsNaN(rowAggregates[i]) {
						rowAggregates[i] = floatVal
					}
				case "max":
					if rowAggregates[i] < floatVal || math.IsNaN(rowAggregates[i]) {
						rowAggregates[i] = floatVal
					}
				}
			}

			rows[rowKey] = rowAggregates
		}

		if len(desc.Columns) > 0 {
			updateRows(rowKey, summaryRows)
		}

		if desc.PointSize > 0 {
			timeGroup := ts / desc.PointSize
			var rows map[string][]float64
			var ok bool
			if rows, ok = summaryRowsByTime[timeGroup]; !ok {
				rows = map[string][]float64{}
				summaryRowsByTime[timeGroup] = rows
			}
			updateRows(rowKey, rows)
		}
	} // Event cursor loop

	if err = cur.Err(); err != nil {
		return nil, err
	}

	summaryEvents := []Event{}
	for rowKey, rowAggregates := range summaryRows {
		event := Event{}
		if len(desc.GroupBy) > 0 {
			parts := strings.Split(rowKey, "\x00")
			for i, part := range parts {
				if desc.GroupBy[i].Name == "_ts" { // TODO: support aggregates on grouped columns
					ts, _ := strconv.Atoi(part)
					event["_ts"] = fromMicrosecondTime(int64(ts))
					continue
				}
				var val interface{}
				dec := json.NewDecoder(strings.NewReader(part))
				dec.UseNumber()
				dec.Decode(&val)
				event[desc.GroupBy[i].Name] = val // TODO: support aggregates on grouped columns
			}
		}
		for i, columnDesc := range desc.Columns {
			fieldName := columnDesc.Aggregate + "(" + columnDesc.Name + ")"
			event[fieldName] = rowAggregates[i]
		}
		rowKeyHash := md5.Sum([]byte(rowKey))
		event["_group_id"] = fmt.Sprintf("%x", rowKeyHash[:8])
		summaryEvents = append(summaryEvents, event)
	}

	if len(desc.OrderBy) != 0 {
		orderByColumns := []string{}
		for _, desc := range desc.OrderBy {
			orderByColumns = append(orderByColumns, desc.Name) // TODO: support aggregates on grouped columns
		}
		var ordering sort.Interface = OrderBy{
			columns: orderByColumns,
			events:  summaryEvents,
		}
		if desc.Descending {
			ordering = sort.Reverse(ordering)
		}
		sort.Stable(ordering)
	}

	if desc.Limit > 0 && len(summaryEvents) > desc.Limit {
		summaryEvents = summaryEvents[:desc.Limit]
	}

	seriesEvents := []Event{}
	if desc.PointSize > 0 {
		for ts, rows := range summaryRowsByTime {
			for rowKey, rowAggregates := range rows {
				event := Event{
					"_ts": fromMicrosecondTime(ts * desc.PointSize),
				}
				if len(desc.GroupBy) > 0 {
					parts := strings.Split(rowKey, "\x00")
					for i, part := range parts {
						if desc.GroupBy[i].Name == "_ts" { // TODO: support aggregates on grouped columns
							continue
						}
						var val interface{}
						dec := json.NewDecoder(strings.NewReader(part))
						dec.UseNumber()
						dec.Decode(&val)
						event[desc.GroupBy[i].Name] = val // TODO: support aggregates on grouped columns
					}
				}
				for i, columnDesc := range desc.Columns {
					fieldName := columnDesc.Aggregate + "(" + columnDesc.Name + ")"
					event[fieldName] = rowAggregates[i]
				}
				rowKeyHash := md5.Sum([]byte(rowKey))
				event["_group_id"] = fmt.Sprintf("%x", rowKeyHash[:8])
				seriesEvents = append(seriesEvents, event)
			}
		}

		sort.Sort(ByTimestamp(seriesEvents))
	}

	return &QueryResult{Summary: summaryEvents, Series: seriesEvents, Events: resultEvents, Query: desc}, nil
}

func splitCollectionID(id string) (int64, string, string, error) {
	if len(id) < 1 {
		return 0, "", "", errors.New("invalid ID 2")
	}
	if id[0] != eventKeyPrefix {
		return 0, "", "", errors.New("invalid ID prefix")
	}
	id = id[1:]

	if len(id) < 8+2 {
		return 0, "", "", errors.New("invalid ID 3")
	}

	formattedTs := [8]byte{
		id[0],
		id[1],
		id[2],
		id[3],
		id[4],
		id[5],
		id[6],
		id[7],
	}

	ts := parseTs(formattedTs)

	id = id[8:]

	if id[0] != '|' {
		return 0, "", "", errors.New("invalid ID 4")
	}

	id = id[1:]

	parts := strings.Split(id, "|")

	if len(parts) != 2 {
		return 0, "", "", errors.New("invalid ID 5")
	}

	return ts, parts[0], parts[1], nil
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
