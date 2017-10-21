package main

import (
	"testing"

	"github.com/Cistern/cistern/internal/query"
)

func TestLimit(t *testing.T) {
	ec, err := CreateEventCollection("/tmp/test_cistern_limit.lm2")
	defer ec.col.Destroy()
	if err != nil {
		t.Fatal(err)
	}
	err = ec.StoreEvents(testEvents)
	if err != nil {
		t.Fatal(err)
	}

	const limit = 5
	result, err := ec.Query(query.Desc{
		Limit: limit,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Events) > limit {
		t.Errorf("expected at most %d events but got %d", limit, len(result.Events))
	}
}

func TestFilter(t *testing.T) {
	ec, err := CreateEventCollection("/tmp/test_cistern_filter.lm2")
	defer ec.col.Destroy()
	if err != nil {
		t.Fatal(err)
	}
	err = ec.StoreEvents(testEvents)
	if err != nil {
		t.Fatal(err)
	}

	type testCase struct {
		Filters         []query.Filter
		ExpectedMatches int
	}

	testCases := []testCase{
		{
			Filters: []query.Filter{
				{
					Column:    "source_port",
					Condition: "=",
					Value:     443.0,
				},
			},
			ExpectedMatches: 3,
		},
		{
			Filters: []query.Filter{
				{
					Column:    "source_port",
					Condition: "!=",
					Value:     443.0,
				},
			},
			ExpectedMatches: 4,
		},
		{
			Filters: []query.Filter{
				{
					Column:    "source_port",
					Condition: "<",
					Value:     443.0,
				},
			},
			ExpectedMatches: 1,
		},
		{
			Filters: []query.Filter{
				{
					Column:    "source_port",
					Condition: "<=",
					Value:     443.0,
				},
			},
			ExpectedMatches: 4,
		},
		{
			Filters: []query.Filter{
				{
					Column:    "source_port",
					Condition: ">",
					Value:     443.0,
				},
			},
			ExpectedMatches: 3,
		},
		{
			Filters: []query.Filter{
				{
					Column:    "source_port",
					Condition: ">=",
					Value:     443.0,
				},
			},
			ExpectedMatches: 6,
		},
		{
			Filters: []query.Filter{
				{
					Column:    "dest_address",
					Condition: "matches",
					Value:     "^52.54.+",
				},
			},
			ExpectedMatches: 3,
		},
	}

	for i, tc := range testCases {
		result, err := ec.Query(query.Desc{
			Filters: tc.Filters,
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Events) != tc.ExpectedMatches {
			t.Errorf("expected %d events but got %d for case %d",
				tc.ExpectedMatches, len(result.Events), i)
		}
	}
}
