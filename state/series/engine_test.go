package series

import (
	"os"
	"testing"
)

func TestEngine(t *testing.T) {
	os.RemoveAll("/tmp/series.db")

	engine, err := NewEngine("/tmp/series.db")
	if err != nil {
		t.Fatal(err)
	}

	t.Log(engine.db.Stats())
}
