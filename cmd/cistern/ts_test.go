package main

import (
	"reflect"
	"testing"
	"time"
)

func TestTs(t *testing.T) {
	b := formatTs(1234567890)
	expected := [8]byte{0, 0, 0, 0, 73, 150, 2, 210}
	if !reflect.DeepEqual(b, expected) {
		t.Error("expected", expected, "got", b)
	}

	if parseTs(b) != 1234567890 {
		t.Error("expected", 1234567890, "got", parseTs(b))
	}
}

func TestMicrosecondTime(t *testing.T) {
	timeTs := time.Unix(1483062681, 1234000).UTC()
	ts := toMicrosecondTime(timeTs)
	if ts != 1483062681001234 {
		t.Error("expected", 1483062681001234, "got", ts)
	}

	timeTsFromMicrosecondTime := fromMicrosecondTime(ts)
	if timeTsFromMicrosecondTime != timeTs {
		t.Error("expected", timeTs, "got", timeTsFromMicrosecondTime)
	}
}
