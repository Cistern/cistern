package main

import (
	"encoding/json"
	"testing"
)

func TestQuerySyntax(t *testing.T) {
	var cases = [][]string{
		{"count(_id)", "interface , foo", "", ""},
		{"sum(bytes), sum(packets)", "source_address, dest_address", "source_address != 172.31.28.236", ""},
		{"", "", "", ""},
	}

	for _, testCase := range cases {
		result, err := parseQuery(testCase[0], testCase[1], testCase[2], testCase[3])
		if err != nil {
			t.Error(err)
		} else {
			marshaled, _ := json.Marshal(result)
			t.Logf("%s", marshaled)
		}
	}
}
