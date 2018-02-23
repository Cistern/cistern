package lm2

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type TxCollection struct {
	col        *Collection
	writerLock sync.Mutex
}

func NewTxCollection(col *Collection) *TxCollection {
	return &TxCollection{
		col:        col,
		writerLock: sync.Mutex{},
	}
}

func (c *TxCollection) View(f func(*Cursor) error) error {
	cursor, err := c.col.NewCursor()
	if err != nil {
		return err
	}
	return f(cursor)
}

func (c *TxCollection) Update(f func(*Cursor, *WriteBatch) error) error {
	c.writerLock.Lock()
	defer c.writerLock.Unlock()

	cursor, err := c.col.NewCursor()
	if err != nil {
		return err
	}
	wb := NewWriteBatch()
	err = f(cursor, wb)
	if err != nil {
		return err
	}
	_, err = c.col.Update(wb)
	return err
}

func TestTransactional1(t *testing.T) {
	col, err := NewCollection("/tmp/test_transactional_1.lm2", 10000)
	if err != nil {
		t.Fatal(err)
	}

	txCol := NewTxCollection(col)

	err = txCol.Update(func(cur *Cursor, wb *WriteBatch) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	err = txCol.View(func(cur *Cursor) error {
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func verifySquares(cur *Cursor) error {
	cur.Seek("a")
	ints := []int{}
	pairs := []string{}
	for cur.Next() {
		pairs = append(pairs, cur.Key()+" => "+cur.Value())
		val := cur.Value()
		n, err := strconv.Atoi(val)
		if err != nil {
			return err
		}
		ints = append(ints, n)
	}
	if cur.Err() != nil {
		return nil
	}

	prev := 0
	for _, n := range ints {
		if prev == 0 {
			prev = n
			continue
		}
		if n != prev*prev {
			return fmt.Errorf("not a square (%d != %d * %d == %d): %v, %v",
				n, prev, prev, prev*prev, ints, pairs)
		}
		prev = n
	}

	return nil
}

func setSquares(wb *WriteBatch) {
	first := rand.Intn(1000)
	second := first * first
	third := second * second
	wb.Set("a", strconv.Itoa(first))
	wb.Set("b", strconv.Itoa(second))
	wb.Set("c", strconv.Itoa(third))
}

func TestTransactionalSquares(t *testing.T) {
	seed := time.Now().Unix()
	t.Log("seed:", seed)
	rand.Seed(seed)
	col, err := NewCollection("/tmp/test_transactional_squares.lm2", 10000)
	if err != nil {
		t.Fatal(err)
	}

	errRandomFailure := errors.New("random failure")

	const failureProbConst = 0.01
	failureProb := failureProbConst
	col.readAt = func(b []byte, off int64) (int, error) {
		if rand.Float64() <= failureProb {
			return 0, errRandomFailure
		}
		return col.f.ReadAt(b, off)
	}

	const count = 100
	const parallelism = 4

	txCol := NewTxCollection(col)

	wg := sync.WaitGroup{}

	expectedReadFailures := uint32(0)
	for i := 0; i < parallelism; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < count; j++ {
				err := txCol.View(verifySquares)
				if err != nil {
					if err.Error() == "lm2: partial read (random failure)" {
						atomic.AddUint32(&expectedReadFailures, 1)
					} else {
						t.Fatal(err)
					}
				}
			}
		}()
	}

	expectedWriteFailures := uint32(0)
	for i := 0; i < parallelism; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < count; j++ {
				err := txCol.Update(func(cur *Cursor, wb *WriteBatch) error {
					setSquares(wb)
					return nil
				})
				if err != nil {
					if !IsRollbackError(err) && err.Error() != "lm2: partial read (random failure)" {
						t.Fatal(err)
					} else {
						atomic.AddUint32(&expectedWriteFailures, 1)
					}
				}
			}
		}()
	}

	wg.Wait()

	t.Log("Tolerated read failures", expectedReadFailures, "write failures", expectedWriteFailures)
}
