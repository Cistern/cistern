package lm2

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"sync/atomic"
)

const sentinelMagic = 0xDEAD10CC

const (
	maxLevels = 4
	levelProb = 0.1
	cacheProb = 0.1
)

var (
	// ErrDoesNotExist is returned when a collection's data file
	// doesn't exist.
	ErrDoesNotExist = errors.New("lm2: does not exist")
	// ErrInternal is returned when the internal state of the collection
	// is invalid. The collection should be closed and reopened.
	ErrInternal = errors.New("lm2: internal error")
	// ErrKeyNotFound is returned when a Cursor.Get() doesn't find
	// the requested key.
	ErrKeyNotFound = errors.New("lm2: key not found")

	fileVersion = [8]byte{'l', 'm', '2', '_', '0', '0', '1', '\n'}
)

// RollbackError is the error type returned after rollbacks.
type RollbackError struct {
	DuplicateKey  bool
	ConflictedKey string
	Err           error
}

func (e RollbackError) Error() string {
	if e.DuplicateKey {
		return fmt.Sprintf("lm2: rolled back due to duplicate key (conflicted key: `%s`)",
			e.ConflictedKey)
	}
	return fmt.Sprintf("lm2: rolled back (%s)", e.Err.Error())
}

// IsRollbackError returns true if err is a RollbackError.
func IsRollbackError(err error) bool {
	_, ok := err.(RollbackError)
	return ok
}

// Collection represents an ordered linked list map.
type Collection struct {
	fileHeader
	f         *os.File
	wal       *wal
	stats     Stats
	dirty     map[int64]*record
	cache     *recordCache
	dirtyLock sync.Mutex

	// internalState is 0 if OK, 1 if inconsistent.
	internalState uint32

	metaLock  sync.RWMutex
	writeLock sync.Mutex

	readAt  func(b []byte, off int64) (n int, err error)
	writeAt func(b []byte, off int64) (n int, err error)
}

type fileHeader struct {
	Version    [8]byte
	Next       [maxLevels]int64
	LastCommit int64
}

func (h fileHeader) bytes() []byte {
	buf := bytes.NewBuffer(nil)
	binary.Write(buf, binary.LittleEndian, h)
	return buf.Bytes()
}

type recordHeader struct {
	_       uint8 // reserved
	_       uint8 // reserved
	Next    [maxLevels]int64
	Deleted int64
	KeyLen  uint16
	ValLen  uint32
}

const recordHeaderSize = 2 + (maxLevels * 8) + 8 + 2 + 4

func (h recordHeader) bytes() []byte {
	buf := bytes.NewBuffer(nil)
	binary.Write(buf, binary.LittleEndian, h)
	return buf.Bytes()
}

type sentinelRecord struct {
	Magic  uint32 // some fixed pattern
	Offset int64  // this record's offset
}

type record struct {
	recordHeader
	Offset int64
	Key    string
	Value  string

	lock sync.RWMutex
}

func generateLevel() int {
	level := 0
	for i := 0; i < maxLevels-1; i++ {
		if rand.Float32() <= levelProb {
			level++
		} else {
			break
		}
	}
	return level
}

func (c *Collection) getDirty(offset int64) *record {
	c.dirtyLock.Lock()
	defer c.dirtyLock.Unlock()
	if c.dirty == nil {
		return nil
	}
	return c.dirty[offset]
}

func (c *Collection) setDirty(offset int64, rec *record) {
	c.dirtyLock.Lock()
	defer c.dirtyLock.Unlock()
	c.dirty[offset] = rec
}

func (c *Collection) readRecord(offset int64, dirty bool) (*record, error) {
	if offset == 0 {
		return nil, errors.New("lm2: invalid record offset 0")
	}

	if dirty {
		if rec := c.getDirty(offset); rec != nil {
			return rec, nil
		}
	}

	c.cache.lock.RLock()
	if rec := c.cache.cache[offset]; rec != nil {
		c.cache.lock.RUnlock()
		c.stats.incRecordsRead(1)
		c.stats.incCacheHits(1)
		return rec, nil
	}
	c.cache.lock.RUnlock()

	recordHeaderBytes := [recordHeaderSize]byte{}
	n, err := c.readAt(recordHeaderBytes[:], offset)
	if err != nil && n != recordHeaderSize {
		return nil, fmt.Errorf("lm2: partial read (%s)", err)
	}

	header := recordHeader{}
	err = binary.Read(bytes.NewReader(recordHeaderBytes[:]), binary.LittleEndian, &header)
	if err != nil {
		return nil, err
	}

	keyValBuf := make([]byte, int(header.KeyLen)+int(header.ValLen))
	n, err = c.readAt(keyValBuf, offset+recordHeaderSize)
	if err != nil && n != len(keyValBuf) {
		return nil, fmt.Errorf("lm2: partial read (%s)", err)
	}

	key := string(keyValBuf[:int(header.KeyLen)])
	value := string(keyValBuf[int(header.KeyLen):])

	rec := &record{
		recordHeader: header,
		Offset:       offset,
		Key:          key,
		Value:        value,
	}
	c.stats.incRecordsRead(1)
	c.stats.incCacheMisses(1)
	c.cache.push(rec)
	return rec, nil
}

func (c *Collection) nextRecord(rec *record, level int, dirty bool) (*record, error) {
	if rec == nil {
		return nil, errors.New("lm2: invalid record")
	}
	if atomic.LoadInt64(&rec.Next[level]) == 0 {
		// There's no next record.
		return nil, nil
	}
	nextRec, err := c.readRecord(atomic.LoadInt64(&rec.Next[level]), dirty)
	if err != nil {
		return nil, err
	}
	return nextRec, nil
}

// NewCollection creates a new collection with a data file at file.
// cacheSize represents the size of the collection cache.
func NewCollection(file string, cacheSize int) (*Collection, error) {
	f, err := os.OpenFile(file, os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		return nil, err
	}
	err = f.Truncate(0)
	if err != nil {
		f.Close()
		return nil, err
	}

	wal, err := newWAL(file + ".wal")
	if err != nil {
		f.Close()
		return nil, err
	}
	c := &Collection{
		f:       f,
		wal:     wal,
		cache:   newCache(cacheSize),
		readAt:  f.ReadAt,
		writeAt: f.WriteAt,
	}

	// write file header
	c.fileHeader.Version = fileVersion
	c.fileHeader.Next[0] = 0
	c.fileHeader.LastCommit = int64(512)
	c.f.Seek(0, 0)
	err = binary.Write(c.f, binary.LittleEndian, c.fileHeader)
	if err != nil {
		c.f.Close()
		c.wal.Close()
		return nil, err
	}
	return c, nil
}

// OpenCollection opens a collection with a data file at file.
// cacheSize represents the size of the collection cache.
// ErrDoesNotExist is returned if file does not exist.
func OpenCollection(file string, cacheSize int) (*Collection, error) {
	f, err := os.OpenFile(file, os.O_RDWR, 0666)
	if err != nil {
		if os.IsNotExist(err) {
			// Check if there's a compacted version.
			if _, err = os.Stat(file + ".compact"); err == nil {
				// There is.
				err = os.Rename(file+".compact", file)
				if err != nil {
					return nil, fmt.Errorf("lm2: error recovering compacted data file: %v", err)
				}
				return OpenCollection(file, cacheSize)
			}
			return nil, ErrDoesNotExist
		}
		return nil, fmt.Errorf("lm2: error opening data file: %v", err)
	}
	// Check if there's a compacted version.
	if _, err = os.Stat(file + ".compact"); err == nil {
		// There is. Remove it and its wal.
		os.Remove(file + ".compact")
		os.Remove(file + ".compact.wal")
	}

	wal, err := openWAL(file + ".wal")
	if os.IsNotExist(err) {
		wal, err = newWAL(file + ".wal")
	}
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("lm2: error WAL: %v", err)
	}

	c := &Collection{
		f:       f,
		wal:     wal,
		cache:   newCache(cacheSize),
		readAt:  f.ReadAt,
		writeAt: f.WriteAt,
	}

	// Read file header.
	c.f.Seek(0, 0)
	err = binary.Read(c.f, binary.LittleEndian, &c.fileHeader)
	if err != nil {
		c.Close()
		return nil, fmt.Errorf("lm2: error reading file header: %v", err)
	}

	// Read last WAL entry.
	lastEntry, err := c.wal.ReadLastEntry()
	if err != nil {
		// Maybe latest WAL write didn't succeed.
		// Truncate.
		c.wal.Truncate()
	} else {
		// Apply last WAL entry again.
		for _, walRec := range lastEntry.records {
			_, err := c.writeAt(walRec.Data, walRec.Offset)
			if err != nil {
				c.Close()
				return nil, fmt.Errorf("lm2: partial write (%s)", err)
			}
		}

		// Reread file header because it could have been updated
		c.f.Seek(0, 0)
		err = binary.Read(c.f, binary.LittleEndian, &c.fileHeader)
		if err != nil {
			c.Close()
			return nil, fmt.Errorf("lm2: error reading file header: %v", err)
		}
	}

	c.f.Truncate(c.LastCommit)

	err = c.sync()
	if err != nil {
		c.Close()
		return nil, err
	}

	return c, nil
}

func (c *Collection) sync() error {
	if err := c.wal.f.Sync(); err != nil {
		return errors.New("lm2: error syncing WAL")
	}
	if err := c.f.Sync(); err != nil {
		return errors.New("lm2: error syncing data file")
	}
	return nil
}

// Close closes a collection and all of its resources.
func (c *Collection) Close() {
	c.metaLock.Lock()
	defer c.metaLock.Unlock()
	c.f.Close()
	c.wal.Close()
	if atomic.LoadUint32(&c.internalState) == 0 {
		// Internal state is OK. Safe to delete WAL.
		c.wal.Destroy()
	}
	atomic.StoreUint32(&c.internalState, 1)
}

// Version returns the last committed version.
func (c *Collection) Version() int64 {
	c.metaLock.RLock()
	defer c.metaLock.RUnlock()
	return c.LastCommit
}

// Stats returns collection statistics.
func (c *Collection) Stats() Stats {
	return c.stats.clone()
}

// Destroy closes the collection and removes its associated data files.
func (c *Collection) Destroy() error {
	c.Close()
	var err error
	err = os.Remove(c.f.Name())
	if err != nil {
		return err
	}
	return nil
}

// Compact rewrites a collection to clean up deleted records and optimize
// data layout on disk.
// NOTE: The collection is closed after compaction, so you'll have to reopen it.
func (c *Collection) Compact() error {
	return c.CompactFunc(func(key, value string) (string, string, bool) {
		return key, value, true
	})
}

// CompactFunc compacts with a custom compaction function. f is called with
// each key-value pair, and it should return the new key and value for that record
// if they should be changed, and whether to keep the record.
// Returning false will skip the record.
// NOTE: The collection is closed after compaction, so you'll have to reopen it.
func (c *Collection) CompactFunc(f func(key, value string) (string, string, bool)) error {
	c.writeLock.Lock()
	defer c.writeLock.Unlock()
	newCollection, err := NewCollection(c.f.Name()+".compact", 10)
	if err != nil {
		return err
	}
	cur, err := c.NewCursor()
	if err != nil {
		return err
	}
	const batchSize = 1000
	remaining := batchSize
	wb := NewWriteBatch()
	for cur.Next() {
		key, val, keep := f(cur.Key(), cur.Value())
		if !keep {
			continue
		}
		wb.Set(key, val)
		remaining--

		if remaining == 0 {
			_, err := newCollection.Update(wb)
			if err != nil {
				return err
			}
			remaining = batchSize
			wb = NewWriteBatch()
		}
	}
	if remaining < batchSize {
		_, err := newCollection.Update(wb)
		if err != nil {
			return err
		}
	}
	err = c.Destroy()
	if err != nil {
		return err
	}
	newCollection.Close()
	return os.Rename(newCollection.f.Name(), c.f.Name())
}

// OK returns true if the internal state of the collection is valid.
// If false is returned you should close and reopen the collection.
func (c *Collection) OK() bool {
	return atomic.LoadUint32(&c.internalState) == 0
}
