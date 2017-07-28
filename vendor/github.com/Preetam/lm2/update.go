package lm2

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sort"
	"sync/atomic"
)

func writeRecord(rec *record, buf *bytes.Buffer) error {
	rec.KeyLen = uint16(len(rec.Key))
	rec.ValLen = uint32(len(rec.Value))

	err := binary.Write(buf, binary.LittleEndian, rec.recordHeader)
	if err != nil {
		return err
	}

	_, err = buf.WriteString(rec.Key)
	if err != nil {
		return err
	}
	_, err = buf.WriteString(rec.Value)
	if err != nil {
		return err
	}

	return nil
}

func (c *Collection) writeSentinel() (int64, error) {
	offset, err := c.f.Seek(0, 2)
	if err != nil {
		return 0, err
	}
	sentinel := sentinelRecord{
		Magic:  sentinelMagic,
		Offset: offset,
	}
	err = binary.Write(c.f, binary.LittleEndian, sentinel)
	if err != nil {
		return 0, err
	}
	return offset + 12, nil
}

func (c *Collection) findLastLessThanOrEqual(key string, startingOffset int64, level int, equal bool) (int64, error) {
	offset := startingOffset

	headOffset := atomic.LoadInt64(&c.Next[level])
	if headOffset == 0 {
		// Empty collection.
		return 0, nil
	}

	var rec *record
	var err error
	if offset == 0 {
		// read the head
		rec, err = c.readRecord(headOffset)
		if err != nil {
			return 0, err
		}
		if rec.Key > key { // we have a new head
			return 0, nil
		}

		if level == maxLevels-1 {
			cacheResult := c.cache.findLastLessThan(key)
			if cacheResult != 0 {
				rec, err = c.readRecord(cacheResult)
				if err != nil {
					return 0, err
				}
			}
		}

		offset = rec.Offset
	} else {
		rec, err = c.readRecord(offset)
		if err != nil {
			return 0, err
		}
	}

	for rec != nil {
		rec.lock.RLock()
		if (!equal && rec.Key == key) || rec.Key > key {
			rec.lock.RUnlock()
			break
		}
		offset = rec.Offset
		oldRec := rec
		rec, err = c.nextRecord(oldRec, level)
		if err != nil {
			return 0, err
		}
		oldRec.lock.RUnlock()
	}

	return offset, nil
}

// Update atomically and durably applies a WriteBatch (a set of updates) to the collection.
// It returns the new version (on success) and an error.
func (c *Collection) Update(wb *WriteBatch) (int64, error) {
	c.writeLock.Lock()
	defer c.writeLock.Unlock()

	if atomic.LoadUint32(&c.internalState) != 0 {
		return 0, ErrInternal
	}

	c.metaLock.Lock()
	defer c.metaLock.Unlock()

	c.dirtyLock.Lock()
	c.dirty = map[int64]*record{}
	c.dirtyLock.Unlock()
	defer func() {
		c.dirtyLock.Lock()
		c.dirty = nil
		c.dirtyLock.Unlock()
	}()

	// Clean up WriteBatch.
	wb.cleanup()

	// Find and load records that will be modified into the cache.

	mergedSetDeleteKeys := map[string]struct{}{}
	for key := range wb.sets {
		mergedSetDeleteKeys[key] = struct{}{}
	}
	keys := []string{}
	for key := range mergedSetDeleteKeys {
		keys = append(keys, key)
	}

	// Sort keys to be inserted or deleted.
	sort.Strings(keys)

	walEntry := newWALEntry()
	appendBuf := bytes.NewBuffer(nil)
	currentOffset, err := c.f.Seek(0, 2)
	if err != nil {
		atomic.StoreUint32(&c.internalState, 1)
		return 0, errors.New("lm2: couldn't get current file offset")
	}

	overwrittenRecords := []int64{}
	startingOffsets := [maxLevels]int64{}
	for _, key := range keys {
		value := wb.sets[key]
		level := generateLevel()
		newRecordOffset := currentOffset + int64(appendBuf.Len())
		rec := &record{
			recordHeader: recordHeader{
				Next: [maxLevels]int64{},
			},
			Offset: newRecordOffset,
			Key:    key,
			Value:  value,
		}

		for i := maxLevels - 1; i > level; i-- {
			offset, err := c.findLastLessThanOrEqual(key, startingOffsets[i], i, true)
			if err != nil {
				return 0, err
			}
			if offset > 0 {
				startingOffsets[i] = offset
				if i > 0 {
					startingOffsets[i-1] = offset
				}
			}
		}

		for ; level >= 0; level-- {
			offset, err := c.findLastLessThanOrEqual(key, startingOffsets[level], level, true)
			if err != nil {
				return 0, err
			}
			if offset == 0 {
				// Insert at head
				atomic.StoreInt64(&rec.Next[level], c.fileHeader.Next[level])
				atomic.StoreInt64(&c.fileHeader.Next[level], newRecordOffset)
			} else {
				// Have a previous record
				var prevRec *record
				if prev := c.getDirty(offset); prev != nil {
					prevRec = prev
				} else {
					prevRec, err = c.readRecord(offset)
					if err != nil {
						atomic.StoreUint32(&c.internalState, 1)
						return 0, err
					}
				}
				atomic.StoreInt64(&rec.Next[level], prevRec.Next[level])
				atomic.StoreInt64(&prevRec.Next[level], newRecordOffset)
				c.setDirty(prevRec.Offset, prevRec)
				walEntry.Push(newWALRecord(prevRec.Offset, prevRec.recordHeader.bytes()))

				if prevRec.Key == key && prevRec.Deleted == 0 {
					overwrittenRecords = append(overwrittenRecords, prevRec.Offset)
				}

				if level > 0 {
					startingOffsets[level-1] = prevRec.Offset
				}
			}

			startingOffsets[level] = newRecordOffset

			err = writeRecord(rec, appendBuf)
			if err != nil {
				atomic.StoreUint32(&c.internalState, 1)
				return 0, err
			}
			c.setDirty(newRecordOffset, rec)
		}
	}

	_, err = io.Copy(c.f, appendBuf)
	if err != nil {
		atomic.StoreUint32(&c.internalState, 1)
		return 0, fmt.Errorf("lm2: appending records failed (%s)", err)
	}

	// Write sentinel record.

	currentOffset, err = c.writeSentinel()
	if err != nil {
		atomic.StoreUint32(&c.internalState, 1)
		return 0, err
	}

	// fsync data file.
	err = c.f.Sync()
	if err != nil {
		atomic.StoreUint32(&c.internalState, 1)
		return 0, err
	}

	c.dirtyLock.Lock()
	for _, dirtyRec := range c.dirty {
		walEntry.Push(newWALRecord(dirtyRec.Offset, dirtyRec.recordHeader.bytes()))
	}
	c.dirtyLock.Unlock()

	for key := range wb.deletes {
		offset := int64(0)
		for level := maxLevels - 1; level >= 0; level-- {
			offset, err = c.findLastLessThanOrEqual(key, offset, level, true)
			if err != nil {
				atomic.StoreUint32(&c.internalState, 1)
				return 0, err
			}
		}
		if offset == 0 {
			continue
		}
		rec, err := c.readRecord(offset)
		if err != nil {
			atomic.StoreUint32(&c.internalState, 1)
			return 0, err
		}
		if rec.Key != key {
			continue
		}
		rec.lock.Lock()
		rec.Deleted = currentOffset
		rec.lock.Unlock()
		walEntry.Push(newWALRecord(rec.Offset, rec.recordHeader.bytes()))
	}

	for _, offset := range overwrittenRecords {
		var rec *record
		if dirtyRec := c.getDirty(offset); dirtyRec != nil {
			rec = dirtyRec
		} else {
			rec, err = c.readRecord(offset)
			if err != nil {
				atomic.StoreUint32(&c.internalState, 1)
				return 0, err
			}
		}
		rec.lock.Lock()
		rec.Deleted = currentOffset
		rec.lock.Unlock()
		walEntry.Push(newWALRecord(rec.Offset, rec.recordHeader.bytes()))
	}

	// ^ record changes should have been serialized + buffered. Write those entries
	// out to the WAL.
	c.LastCommit = currentOffset
	walEntry.Push(newWALRecord(0, c.fileHeader.bytes()))
	_, err = c.wal.Append(walEntry)
	if err != nil {
		atomic.StoreUint32(&c.internalState, 1)
		return 0, err
	}

	// Update + fsync data file header.
	for _, walRec := range walEntry.records {
		_, err := c.writeAt(walRec.Data, walRec.Offset)
		if err != nil {
			atomic.StoreUint32(&c.internalState, 1)
			return 0, fmt.Errorf("lm2: partial write (%s)", err)
		}
	}

	err = c.f.Sync()
	if err != nil {
		atomic.StoreUint32(&c.internalState, 1)
		return 0, err
	}

	return c.LastCommit, nil
}
