package main

import "testing"

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
	result, err := ec.Query(QueryDesc{
		Limit: limit,
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Events) > limit {
		t.Errorf("expected at most %d events but got %d", limit, len(result.Events))
	}
}
