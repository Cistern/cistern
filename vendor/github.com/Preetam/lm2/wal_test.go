package lm2

import (
	"bytes"
	"testing"
)

func TestWAL(t *testing.T) {
	wal, err := newWAL("/tmp/test.wal")
	if err != nil {
		t.Fatal(err)
	}
	entry := newWALEntry()
	record := newWALRecord(4321, []byte("test record"))
	entry.Push(record)
	if entry.NumRecords != 1 {
		t.Errorf("expected entry.NumRecords to be %d, got %d", 1, entry.NumRecords)
	}
	_, err = wal.Append(entry)
	if err != nil {
		t.Error(err)
	}
	wal.Close()

	wal, err = openWAL("/tmp/test.wal")
	if err != nil {
		t.Fatal(err)
	}
	defer wal.Destroy()
	readEntry, err := wal.readEntry()
	if err != nil {
		t.Fatal(err)
	}
	if len(readEntry.records) != 1 {
		t.Fatalf("expected %d record, got %d", 1, len(readEntry.records))
	}
	rec := readEntry.records[0]
	if rec.Offset != 4321 {
		t.Errorf("expected offset %d, got %d", 4321, rec.Offset)
	}
	if !bytes.Equal(rec.Data, []byte("test record")) {
		t.Errorf("expected record data %v, got %v", []byte("test record"), rec.Data)
	}
}
