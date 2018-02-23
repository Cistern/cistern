package lm2

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func verifyOrder(t *testing.T, c *Collection, errLock *sync.Mutex) int {
	count := 0
	prev := ""
	cur, err := c.NewCursor()
	if err != nil {
		if errLock != nil {
			errLock.Lock()
		}
		t.Fatal(err)
		if errLock != nil {
			errLock.Unlock()
		}
	}
	for cur.Next() {
		count++
		if cur.Key() < prev {
			t.Errorf("key %v greater than previous key %v", cur.Key(), prev)
		}
	}
	if err = cur.Err(); err != nil {
		if errLock != nil {
			errLock.Lock()
		}
		t.Fatal(err)
		if errLock != nil {
			errLock.Unlock()
		}
	}
	return count
}

func TestCopy(t *testing.T) {
	c, err := NewCollection("/tmp/test_copy.lm2", 100)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Destroy()

	const failureProbConst = 0.001
	failureProb := failureProbConst
	c.readAt = func(b []byte, off int64) (int, error) {
		if rand.Float64() <= failureProb {
			return 0, errors.New("random failure")
		}
		return c.f.ReadAt(b, off)
	}

	const N = 1000
	firstWriteStart := time.Now()
	for i := 0; i < N; i++ {
		key := fmt.Sprintf("%019d-%019d-%019d-%019d-%019d-%019d-%019d-%019d",
			rand.Int63(), rand.Int63(), rand.Int63(), rand.Int63(),
			rand.Int63(), rand.Int63(), rand.Int63(), rand.Int63())
		val := fmt.Sprint(i)
		wb := NewWriteBatch()
		wb.Set(key, val)
	RETRY:
		if _, err := c.Update(wb); err != nil {
			if IsRollbackError(err) {
				t.Log("rollback")
				goto RETRY
			}
			t.Fatal(err)
		}
	}
	t.Log("First write pass time:", time.Now().Sub(firstWriteStart))
	failureProb = 0
	verifyOrder(t, c, nil)

	c2, err := NewCollection("/tmp/test_copy_copy.lm2", 100)
	if err != nil {
		t.Fatal(err)
	}
	defer c2.Destroy()

	cur, err := c.NewCursor()
	if err != nil {
		t.Fatal(err)
	}

	work := make(chan [2]string, 100)
	go func() {
		for cur.Next() {
			work <- [2]string{cur.Key(), cur.Value()}
		}
		if err = cur.Err(); err != nil {
			t.Fatal(err)
		}
		close(work)
	}()

	secondWriteStart := time.Now()
	const batchSize = 100
	remaining := batchSize
	wb := NewWriteBatch()
	for pair := range work {
		wb.Set(pair[0], pair[1])
		remaining--

		if remaining == 0 {
		RETRY2:
			_, err := c2.Update(wb)
			if err != nil {
				if IsRollbackError(err) {
					t.Log("rollback")
					goto RETRY2
				}
				t.Fatal(err)
			}
			remaining = batchSize
			wb = NewWriteBatch()
		}
	}

	if remaining < batchSize {
	RETRY3:
		_, err := c2.Update(wb)
		if err != nil {
			if IsRollbackError(err) {
				t.Log("rollback")
				goto RETRY3
			}
			t.Fatal(err)
		}
	}

	t.Log("Second write pass time:", time.Now().Sub(secondWriteStart))

	firstStart := time.Now()
	failureProb = 0
	count1 := verifyOrder(t, c, nil)
	firstEnd := time.Now()
	secondStart := firstEnd
	count2 := verifyOrder(t, c2, nil)
	secondEnd := time.Now()
	t.Log("Time to iterate through first list:", firstEnd.Sub(firstStart), "with", count1, "elements")
	t.Log("Time to iterate through second list:", secondEnd.Sub(secondStart), "with", count2, "elements")

	if count1 != count2 || count1 != N {
		t.Errorf("incorrect count. N = %d, count1 = %d, count2 = %d", N, count1, count2)
	}
	t.Logf("%+v", c.Stats())
	t.Logf("%+v", c2.Stats())
}

func TestWriteBatch(t *testing.T) {
	expected := [][2]string{
		{"key1", "1"},
		{"key2", "2"},
		{"key3", "3"},
		{"key4", "4"},
	}

	c, err := NewCollection("/tmp/test_writebatch.lm2", 100)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Destroy()

	wb := NewWriteBatch()
	wb.Set("key1", "1")
	wb.Set("key2", "2")
	wb.Set("key3", "3")
	wb.Set("key4", "4")

	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	verifyOrder(t, c, nil)

	cur, err := c.NewCursor()
	if err != nil {
		t.Fatal(err)
	}

	i := 0
	for cur.Next() {
		if i == len(expected) {
			t.Fatal("unexpected key", cur.Key())
		}
		if cur.Key() != expected[i][0] || cur.Value() != expected[i][1] {
			t.Errorf("expected %v => %v, got %v => %v",
				expected[i][0], expected[i][1], cur.Key(), cur.Value())
		}
		i++
	}
	if err = cur.Err(); err != nil {
		t.Fatal(err)
	}

	expected = [][2]string{
		{"key1", "5"},
		{"key2", "6"},
		{"key3", "7"},
		{"key4", "8"},
	}

	wb = NewWriteBatch()
	wb.Set("key1", "5")
	wb.Set("key2", "6")
	wb.Set("key3", "7")
	wb.Set("key4", "8")

	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	verifyOrder(t, c, nil)

	cur, err = c.NewCursor()
	if err != nil {
		t.Fatal(err)
	}

	i = 0
	for cur.Next() {
		if cur.current.Deleted > 0 {
			continue
		}
		if i == len(expected) {
			t.Fatal("unexpected key", cur.Key())
		}
		if cur.Key() != expected[i][0] || cur.Value() != expected[i][1] {
			t.Errorf("expected %v => %v, got %v => %v",
				expected[i][0], expected[i][1], cur.Key(), cur.Value())
			t.Logf("%+#v", cur.current)
		}
		i++
	}
	if err = cur.Err(); err != nil {
		t.Fatal(err)
	}

	// Check if cursor can be reset
	cur.Seek("")
	i = 0
	for cur.Next() {
		if cur.current.Deleted > 0 {
			continue
		}
		if i == len(expected) {
			t.Fatal("unexpected key", cur.Key())
		}
		if cur.Key() != expected[i][0] || cur.Value() != expected[i][1] {
			t.Errorf("expected %v => %v, got %v => %v",
				expected[i][0], expected[i][1], cur.Key(), cur.Value())
			t.Logf("%+#v", cur.current)
		}
		i++
	}
	if err = cur.Err(); err != nil {
		t.Fatal(err)
	}
}

func TestWriteBatch1(t *testing.T) {
	c, err := NewCollection("/tmp/test_writebatch1.lm2", 100)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Destroy()

	const N = 500
	for i := 0; i < N; i++ {
		wb := NewWriteBatch()
		key := fmt.Sprint(rand.Intn(N * 4))
		val := fmt.Sprint(i)
		wb.Set(key, val)
		if _, err := c.Update(wb); err != nil {
			t.Fatal(err)
		}
	}
	verifyOrder(t, c, nil)
}

func TestWriteBatch1Concurrent(t *testing.T) {
	c, err := NewCollection("/tmp/test_writebatch1concurrent.lm2", 100)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Destroy()

	const N = 50
	const NumGoroutines = 8

	startWG := sync.WaitGroup{}
	endWG := sync.WaitGroup{}
	startWG.Add(NumGoroutines)
	endWG.Add(NumGoroutines)

	errLock := &sync.Mutex{}

	for i := 0; i < NumGoroutines; i++ {
		go func() {
			for j := 0; j < N; j++ {
				wb := NewWriteBatch()
				key := fmt.Sprint(rand.Intn(N * 4))
				val := fmt.Sprint(j)
				wb.Set(key, val)
				if _, err := c.Update(wb); err != nil {
					errLock.Lock()
					t.Fatal(err)
					errLock.Unlock()
				}
				if j == 0 {
					startWG.Done()
				}
			}
			endWG.Done()
		}()
	}

	// Wait for them to start working.
	startWG.Wait()

	for i := 0; i < 1000; i++ {
		verifyOrder(t, c, errLock)
		time.Sleep(time.Millisecond)
	}

	// Wait for them to end.
	endWG.Wait()

	verifyOrder(t, c, errLock)
}

func TestWriteBatch2(t *testing.T) {
	expected := [][2]string{
		{"key1", "1"},
		{"key2", "2"},
		{"key3", "1"},
	}

	c, err := NewCollection("/tmp/test_writebatch2.lm2", 100)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Destroy()

	wb := NewWriteBatch()

	wb.Set("key1", "1")
	t.Log("Set", "key1", "1")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	wb = NewWriteBatch()
	wb.Set("key2", "1")
	t.Log("Set", "key2", "1")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	wb = NewWriteBatch()
	wb.Set("key3", "1")
	t.Log("Set", "key3", "1")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	wb = NewWriteBatch()
	wb.Set("key2", "2")
	t.Log("Set", "key2", "2")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	wb = NewWriteBatch()
	wb.Set("key4", "1")
	t.Log("Set", "key4", "1")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	wb = NewWriteBatch()
	wb.Delete("key4")
	t.Log("Delete", "key4")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	verifyOrder(t, c, nil)

	cur, err := c.NewCursor()
	if err != nil {
		t.Fatal(err)
	}

	i := 0
	for cur.Next() {
		if i == len(expected) {
			t.Fatal("unexpected key", cur.Key())
		}
		if cur.Key() != expected[i][0] || cur.Value() != expected[i][1] {
			t.Errorf("expected %v => %v, got %v => %v",
				expected[i][0], expected[i][1], cur.Key(), cur.Value())
		}
		i++
	}
	if err = cur.Err(); err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", c.Stats())
}

func TestWriteCloseOpen(t *testing.T) {
	expected := [][2]string{
		{"key1", "1"},
		{"key2", "2"},
		{"key3", "1"},
	}

	c, err := NewCollection("/tmp/test_writecloseopen.lm2", 100)
	if err != nil {
		t.Fatal(err)
	}

	wb := NewWriteBatch()
	wb.Set("key1", "1")
	wb.Set("key2", "1")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	wb = NewWriteBatch()
	wb.Set("key3", "1")
	wb.Set("key2", "2")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	wb = NewWriteBatch()
	wb.Set("key4", "1")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	wb = NewWriteBatch()
	wb.Delete("key4")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	c.Close()

	c, err = OpenCollection("/tmp/test_writecloseopen.lm2", 100)
	if err != nil {
		t.Fatal(err)
	}

	verifyOrder(t, c, nil)

	cur, err := c.NewCursor()
	if err != nil {
		t.Fatal(err)
	}

	i := 0
	for cur.Next() {
		if i == len(expected) {
			t.Fatal("unexpected key", cur.Key())
		}
		if cur.Key() != expected[i][0] || cur.Value() != expected[i][1] {
			t.Errorf("expected %v => %v, got %v => %v",
				expected[i][0], expected[i][1], cur.Key(), cur.Value())
		}
		i++
	}
	if err = cur.Err(); err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", c.Stats())

	err = c.Destroy()
	if err != nil {
		t.Fatal(err)
	}
}

func TestSeekToFirstKey(t *testing.T) {
	c, err := NewCollection("/tmp/test_seektofirstkey.lm2", 100)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Destroy()

	wb := NewWriteBatch()
	wb.Set("a", "1")
	wb.Set("b", "1")
	wb.Set("c", "1")
	wb.Set("d", "1")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	cur, err := c.NewCursor()
	if err != nil {
		t.Fatal(err)
	}

	cur.Seek("a")
	if !cur.Valid() {
		t.Fatal("expected cursor to be valid")
	}

	if !cur.Next() {
		t.Fatal("expected Next() to return true")
	}

	if cur.Key() != "a" {
		t.Fatalf("expected cursor key to be 'a', got %v", cur.Key())
	}

	if err = cur.Err(); err != nil {
		t.Fatal(err)
	}
}

func TestOverwriteFirstKey(t *testing.T) {
	c, err := NewCollection("/tmp/test_overwritefirstkey.lm2", 100)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Destroy()

	wb := NewWriteBatch()
	wb.Set("a", "1")
	wb.Set("b", "1")
	wb.Set("c", "1")
	wb.Set("d", "1")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	wb = NewWriteBatch()
	wb.Set("a", "2")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	cur, err := c.NewCursor()
	if err != nil {
		t.Fatal(err)
	}

	cur.Seek("a")
	if !cur.Valid() {
		t.Fatal("expected cursor to be valid")
	}

	if !cur.Next() {
		t.Fatal("expected Next() to return true")
	}

	if cur.Key() != "a" {
		t.Fatalf("expected cursor key to be 'a', got %v", cur.Key())
	}

	if !cur.Next() {
		t.Fatal("expected Next() to return true")
	}

	if cur.Key() != "b" {
		t.Fatalf("expected cursor key to be 'b', got %v", cur.Key())
	}

	if err = cur.Err(); err != nil {
		t.Fatal(err)
	}
}

func TestOverwriteFirstKeyOnly(t *testing.T) {
	c, err := NewCollection("/tmp/test_overwritefirstkeyonly.lm2", 100)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Destroy()

	wb := NewWriteBatch()
	wb.Set("a", "1")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	wb = NewWriteBatch()
	wb.Set("a", "2")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	cur, err := c.NewCursor()
	if err != nil {
		t.Fatal(err)
	}

	cur.Seek("")
	if !cur.Valid() {
		t.Fatal("expected cursor to be valid")
	}

	if !cur.Next() {
		t.Fatal("expected Next() to return true")
	}

	if cur.Key() != "a" {
		t.Fatalf("expected cursor key to be 'a', got %v", cur.Key())
	}
	t.Log(cur.Key(), "=>", cur.Value())

	if cur.Next() {
		t.Error("expected Next() to return false")
		t.Log(cur.Key(), "=>", cur.Value())
	}

	if err = cur.Err(); err != nil {
		t.Fatal(err)
	}
}

func TestDeleteInFirstUpdate(t *testing.T) {
	c, err := NewCollection("/tmp/test_deleteinfirstupdate.lm2", 100)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Destroy()

	wb := NewWriteBatch()
	wb.Delete("a")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSeekOverwrittenKey(t *testing.T) {
	c, err := NewCollection("/tmp/test_seekoverwrittenkey.lm2", 100)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Destroy()

	wb := NewWriteBatch()
	wb.Set("committed", "0")
	wb.Delete("pending")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	wb = NewWriteBatch()
	wb.Set("1", "a")
	wb.Set("pending", "1")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	wb = NewWriteBatch()
	wb.Set("committed", "1")
	wb.Delete("pending")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	cur, err := c.NewCursor()
	if err != nil {
		t.Fatal(err)
	}

	cur.Seek("committed")
	if !cur.Next() {
		t.Fatal("expected cur.Next() to return true")
	}

	if cur.Key() != "committed" {
		t.Fatalf("expected cur.Key() to be %s, got %s", "committed", cur.Key())
	}

	if err = cur.Err(); err != nil {
		t.Fatal(err)
	}
}

func TestEmptyCollectionCursor(t *testing.T) {
	c, err := NewCollection("/tmp/test_emptycollectioncursor.lm2", 100)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Destroy()

	cur, err := c.NewCursor()
	if err != nil {
		t.Fatal(err)
	}

	cur.Seek("asdf")

	if cur.Valid() {
		t.Errorf("expected cur.Valid() to return false")
	}
	for cur.Next() {
		t.Error("expected cur.Next() to return false")
	}
	if cur.Err() != nil {
		t.Error("unexpected error:", cur.Err())
	}
}

func TestSimple(t *testing.T) {
	expected := [][2]string{
		{"key1", "1"},
		{"key2", "2"},
		{"key3", "3"},
	}

	c, err := NewCollection("/tmp/test_simple.lm2", 100)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Destroy()

	wb := NewWriteBatch()
	wb.Set("key2", "2")
	t.Log("Set", "key2", "2")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	wb = NewWriteBatch()
	wb.Set("key1", "1")
	t.Log("Set", "key1", "1")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	wb = NewWriteBatch()
	wb.Set("key3", "3")
	t.Log("Set", "key3", "3")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	verifyOrder(t, c, nil)

	cur, err := c.NewCursor()
	if err != nil {
		t.Fatal(err)
	}

	i := 0
	for cur.Next() {
		if i == len(expected) {
			t.Fatal("unexpected key", cur.Key())
		}
		if cur.Key() != expected[i][0] || cur.Value() != expected[i][1] {
			t.Errorf("expected %v => %v, got %v => %v",
				expected[i][0], expected[i][1], cur.Key(), cur.Value())
		} else {
			t.Logf("got %v => %v", cur.Key(), cur.Value())
		}
		i++
	}
	if err = cur.Err(); err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", c.Stats())
}

func TestLm2Log(t *testing.T) {
	expected := [][2]string{
		{"committed", "0"},
	}

	c, err := NewCollection("/tmp/test_lm2log.lm2", 100)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Destroy()

	wb := NewWriteBatch()
	wb.Set("committed", "0")
	wb.Delete("prepared")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	verifyOrder(t, c, nil)

	cur, err := c.NewCursor()
	if err != nil {
		t.Fatal(err)
	}

	i := 0
	for cur.Next() {
		if i == len(expected) {
			t.Fatal("unexpected key", cur.Key())
		}
		if cur.Key() != expected[i][0] || cur.Value() != expected[i][1] {
			t.Errorf("expected %v => %v, got %v => %v",
				expected[i][0], expected[i][1], cur.Key(), cur.Value())
		} else {
			t.Logf("got %v => %v", cur.Key(), cur.Value())
		}
		i++
	}
	if err = cur.Err(); err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", c.Stats())
}

func TestCompact(t *testing.T) {
	expected := [][2]string{
		{"key1", "a"},
		{"key2", "2"},
		{"key3", "c"},
	}

	c, err := NewCollection("/tmp/test_compact.lm2", 100)
	if err != nil {
		t.Fatal(err)
	}

	wb := NewWriteBatch()
	wb.Set("key2", "2")
	t.Log("Set", "key2", "2")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	wb = NewWriteBatch()
	wb.Set("key1", "1")
	t.Log("Set", "key1", "1")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	wb = NewWriteBatch()
	wb.Set("key3", "3")
	t.Log("Set", "key3", "3")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	wb = NewWriteBatch()
	wb.Set("key1", "a")
	t.Log("Set", "key1", "a")
	wb.Set("key3", "c")
	t.Log("Set", "key3", "c")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	verifyOrder(t, c, nil)

	err = c.Compact()
	if err != nil {
		t.Fatal(err)
	}

	c, err = OpenCollection("/tmp/test_compact.lm2", 100)
	defer c.Destroy()

	cur, err := c.NewCursor()
	if err != nil {
		t.Fatal(err)
	}

	i := 0
	for cur.Next() {
		if i == len(expected) {
			t.Fatal("unexpected key", cur.Key())
		}
		if cur.Key() != expected[i][0] || cur.Value() != expected[i][1] {
			t.Errorf("expected %v => %v, got %v => %v",
				expected[i][0], expected[i][1], cur.Key(), cur.Value())
		} else {
			t.Logf("got %v => %v", cur.Key(), cur.Value())
		}
		i++
	}
	if err = cur.Err(); err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", c.Stats())
}

func TestCopyCompact(t *testing.T) {
	c, err := NewCollection("/tmp/test_copycompact.lm2", 100)
	if err != nil {
		t.Fatal(err)
	}

	const N = 1000
	firstWriteStart := time.Now()
	for i := 0; i < N; i++ {
		key := fmt.Sprintf("%019d-%019d-%019d-%019d-%019d-%019d-%019d-%019d",
			rand.Int63(), rand.Int63(), rand.Int63(), rand.Int63(),
			rand.Int63(), rand.Int63(), rand.Int63(), rand.Int63())
		val := fmt.Sprint(i)
		wb := NewWriteBatch()
		wb.Set(key, val)
		if _, err := c.Update(wb); err != nil {
			t.Fatal(err)
		}
	}
	t.Log("First write pass time:", time.Now().Sub(firstWriteStart))
	verifyOrder(t, c, nil)

	compactStart := time.Now()
	err = c.Compact()
	if err != nil {
		c.Destroy()
		t.Fatal(err)
	}
	t.Log("Compact time:", time.Now().Sub(compactStart))

	c, err = OpenCollection("/tmp/test_copycompact.lm2", 100)
	defer c.Destroy()

	count := verifyOrder(t, c, nil)
	if count != N {
		t.Error("expected count", N, "got", count)
	}
}

func TestCompactSkipKey(t *testing.T) {
	expected := [][2]string{
		{"key1", "a"},
		{"key3", "c"},
	}

	c, err := NewCollection("/tmp/test_compactskip.lm2", 100)
	if err != nil {
		t.Fatal(err)
	}

	wb := NewWriteBatch()
	wb.Set("key2", "2")
	t.Log("Set", "key2", "2")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	wb = NewWriteBatch()
	wb.Set("key1", "1")
	t.Log("Set", "key1", "1")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	wb = NewWriteBatch()
	wb.Set("key3", "3")
	t.Log("Set", "key3", "3")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	wb = NewWriteBatch()
	wb.Set("key1", "a")
	t.Log("Set", "key1", "a")
	wb.Set("key3", "c")
	t.Log("Set", "key3", "c")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	verifyOrder(t, c, nil)

	err = c.CompactFunc(func(key, val string) (string, string, bool) {
		if key == "key2" {
			return "", "", false
		}
		return key, val, true
	})
	if err != nil {
		t.Fatal(err)
	}

	c, err = OpenCollection("/tmp/test_compactskip.lm2", 100)
	defer c.Destroy()

	cur, err := c.NewCursor()
	if err != nil {
		t.Fatal(err)
	}

	i := 0
	for cur.Next() {
		if i == len(expected) {
			t.Fatal("unexpected key", cur.Key())
		}
		if cur.Key() != expected[i][0] || cur.Value() != expected[i][1] {
			t.Errorf("expected %v => %v, got %v => %v",
				expected[i][0], expected[i][1], cur.Key(), cur.Value())
		} else {
			t.Logf("got %v => %v", cur.Key(), cur.Value())
		}
		i++
	}
	if err = cur.Err(); err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", c.Stats())
}

func TestDeleteAndUpdate(t *testing.T) {
	expected := [][2]string{
		{"key1", "1"},
		{"key3", "3"},
	}

	c, err := NewCollection("/tmp/test_deleteandupdate.lm2", 100)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Destroy()

	wb := NewWriteBatch()
	wb.Set("key2", "2")
	t.Log("Set", "key2", "2")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	wb = NewWriteBatch()
	wb.Set("key1", "1")
	t.Log("Set", "key1", "1")
	wb.Set("key3", "3")
	t.Log("Set", "key3", "3")
	wb.Delete("key2")
	t.Log("Delete", "key2")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	verifyOrder(t, c, nil)

	cur, err := c.NewCursor()
	if err != nil {
		t.Fatal(err)
	}

	i := 0
	for cur.Next() {
		if i == len(expected) {
			t.Fatal("unexpected key", cur.Key())
		}
		if cur.Key() != expected[i][0] || cur.Value() != expected[i][1] {
			t.Errorf("expected %v => %v, got %v => %v",
				expected[i][0], expected[i][1], cur.Key(), cur.Value())
		} else {
			t.Logf("got %v => %v", cur.Key(), cur.Value())
		}
		i++
	}
	if err = cur.Err(); err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", c.Stats())
}

func TestOK(t *testing.T) {
	c, err := NewCollection("/tmp/test_ok.lm2", 100)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Destroy()

	c.writeAt = func(b []byte, offset int64) (int, error) {
		return 0, errors.New("some failure")
	}

	wb := NewWriteBatch()
	wb.Set("key2", "2")
	t.Log("Set", "key2", "2")
	_, err = c.Update(wb)
	if err == nil {
		t.Fatal("expected an error")
	}

	if c.internalState != 1 {
		t.Errorf("expected internalState to be 1 but got %d", c.internalState)
	}

	if c.OK() {
		t.Error("expected OK() to return false")
	}
}

func TestConflictRollback(t *testing.T) {
	c, err := NewCollection("/tmp/test_conflictrollback.lm2", 100)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Destroy()

	wb := NewWriteBatch()
	wb.Set("key1", "1")
	t.Log("Set", "key1", "1")
	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	wb = NewWriteBatch()
	wb.AllowOverwrite(false)
	wb.Set("key1", "2")
	t.Log("Set", "key1", "2")
	_, err = c.Update(wb)
	if err == nil {
		t.Fatal("expected a rollback")
	}

	if !IsRollbackError(err) {
		t.Fatal("expected a rollback error")
	}

	rollbackErr := err.(RollbackError)
	if !rollbackErr.DuplicateKey {
		t.Error("expected DuplicateKey to be true")
	}

	if rollbackErr.ConflictedKey != "key1" {
		t.Errorf("expected ConflictedKey to be `%s`, got `%s`",
			"key1", rollbackErr.ConflictedKey)
	}

	if !c.OK() {
		t.Error("expected OK() to return true")
	}
}

func TestCursorGet(t *testing.T) {
	c, err := NewCollection("/tmp/test_cursorget.lm2", 100)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Destroy()

	wb := NewWriteBatch()
	wb.Set("key1", "1")
	wb.Set("key2", "2")
	wb.Set("key3", "3")
	wb.Set("key4", "4")

	_, err = c.Update(wb)
	if err != nil {
		t.Fatal(err)
	}

	verifyOrder(t, c, nil)

	cur, err := c.NewCursor()
	if err != nil {
		t.Fatal(err)
	}

	key2Val, err := cur.Get("key2")
	if err != nil {
		t.Fatal(err)
	}

	if key2Val != "2" {
		t.Fatalf("expected key2Val to be %s but got %s", "2", key2Val)
	}

	key1Val, err := cur.Get("key1")
	if err != nil {
		t.Fatal(err)
	}

	if key1Val != "1" {
		t.Fatalf("expected key1Val to be %s but got %s", "1", key1Val)
	}

	key4Val, err := cur.Get("key4")
	if err != nil {
		t.Fatal(err)
	}

	if key4Val != "4" {
		t.Fatalf("expected key4Val to be %s but got %s", "4", key4Val)
	}

	_, err = cur.Get("missing")
	if err != ErrKeyNotFound {
		t.Fatalf("expected ErrKeyNotFound but got %v", err)
	}
}
