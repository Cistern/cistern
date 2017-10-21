package query

import (
	"encoding/json"
	"reflect"
	"testing"
)

type testCase struct {
	query    string
	expected *Desc
}

func TestParse(t *testing.T) {
	testCases := []testCase{
		{
			query: "SELECT _id",
			expected: &Desc{
				Columns: []ColumnDesc{
					{Name: "_id"},
				},
			},
		},
		{
			query: "SELECT _id, foo",
			expected: &Desc{
				Columns: []ColumnDesc{
					{Name: "_id"},
					{Name: "foo"},
				},
			},
		},
		{
			query: "SELECT sum(bytes)",
			expected: &Desc{
				Columns: []ColumnDesc{
					{Aggregate: "sum", Name: "bytes"},
				},
			},
		},
		{
			query: "SELECT sum(bytes), foo",
			expected: &Desc{
				Columns: []ColumnDesc{
					{Aggregate: "sum", Name: "bytes"},
					{Name: "foo"},
				},
			},
		},
		{
			query: "SELECT sum(bytes) LIMIT 1",
			expected: &Desc{
				Columns: []ColumnDesc{
					{Aggregate: "sum", Name: "bytes"},
				},
				Limit: 1,
			},
		},
		{
			query: "SELECT sum(bytes) LIMIT 1 POINT SIZE 5s",
			expected: &Desc{
				Columns: []ColumnDesc{
					{Aggregate: "sum", Name: "bytes"},
				},
				Limit:     1,
				PointSize: 5000000,
			},
		},
		{
			query: "SELECT sum(bytes) GROUP BY source_addr, dest_addr FILTER a != \"3\" (b = 4.3)(a = 1) ORDER BY source_addr limit 100",
			expected: &Desc{
				Columns: []ColumnDesc{
					{Aggregate: "sum", Name: "bytes"},
				},
				GroupBy: []ColumnDesc{
					{Name: "source_addr"},
					{Name: "dest_addr"},
				},
				Filters: []Filter{
					{Column: "a", Condition: "!=", Value: json.RawMessage(`"3"`)},
					{Column: "b", Condition: "=", Value: json.RawMessage(`4.3`)},
					{Column: "a", Condition: "=", Value: json.RawMessage(`1`)},
				},
				OrderBy: []ColumnDesc{
					{Name: "source_addr"},
				},
				Limit: 100,
			},
		},
		{
			query: "select	sum(bytes) \n group by source_addr, dest_addr filter a != \"3\" (b = 4.3)(a = 1) order by source_addr limit 100 point size 1h",
			expected: &Desc{
				Columns: []ColumnDesc{
					{Aggregate: "sum", Name: "bytes"},
				},
				GroupBy: []ColumnDesc{
					{Name: "source_addr"},
					{Name: "dest_addr"},
				},
				Filters: []Filter{
					{Column: "a", Condition: "!=", Value: json.RawMessage(`"3"`)},
					{Column: "b", Condition: "=", Value: json.RawMessage(`4.3`)},
					{Column: "a", Condition: "=", Value: json.RawMessage(`1`)},
				},
				OrderBy: []ColumnDesc{
					{Name: "source_addr"},
				},
				Limit:     100,
				PointSize: 3600000000,
			},
		},
		{
			query: "SELECT sum(bytes) GROUP BY min(source_addr), dest_addr ORDER BY source_addr, sum(bytes) desc limit 100",
			expected: &Desc{
				Columns: []ColumnDesc{
					{Aggregate: "sum", Name: "bytes"},
				},
				GroupBy: []ColumnDesc{
					{Aggregate: "min", Name: "source_addr"},
					{Name: "dest_addr"},
				},
				OrderBy: []ColumnDesc{
					{Name: "source_addr"},
					{Aggregate: "sum", Name: "bytes"},
				},
				Limit:      100,
				Descending: true,
			},
		},

		// Invalid

		{query: "SELECT"},
		{query: "SELECT 1"},
		{query: "GROUP"},
		{query: "SELECT 1(bytes), foo"},
		{query: "SELECT sum(bytes) LIMIT 1 POINT SIZE -5"},
		{query: "SELECT sum(bytes) LIMIT -1 POINT SIZE -5"},
		{query: "SELECT sum(bytes) GROUP BY min(source_addr), dest_addr ORDER"},
		{query: "SELECT (bytes)"},
	}

	for _, c := range testCases {
		desc, err := Parse(c.query)
		if err != nil {
			if c.expected == nil {
				// Expected an error
				continue
			}
			t.Errorf("Error parsing \"%s\": %s", c.query, err)
			continue
		}
		if !reflect.DeepEqual(desc, c.expected) {
			t.Errorf("%s:\ngot      %v\nexpected %v", c.query, desc, c.expected)
		}
	}
}
