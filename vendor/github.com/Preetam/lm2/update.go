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

func (c *Collection) findLastLessThanOrEqual(key string, startingOffset int64, level int, equal bool, dirty bool) (int64, error) {
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
		rec, err = c.readRecord(headOffset, dirty)
		if err != nil {
			return 0, err
		}
		if rec.Key > key { // we have a new head
			return 0, nil
		}

		if level == maxLevels-1 {
			cacheResult := c.cache.findLastLessThan(key)
			if cacheResult != 0 {
				rec, err = c.readRecord(cacheResult, dirty)
				if err != nil {
					return 0, err
				}
			}
		}

		offset = rec.Offset
	} else {
		rec, err = c.readRecord(offset, dirty)
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
		rec, err = c.nextRecord(oldRec, level, dirty)
		if err != nil {
			return 0, err
		}
		oldRec.lock.RUnlock()
	}

	return offset, nil
}

// Update atomically and durably applies a WriteBatch (a set of updates) to the collection.
// It returns the new version (on success) and an error.
// The error may be a RollbackError; use IsRollbackError to check.
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
	dirtyOffsets := []int64{}

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

	previousFileHeader := c.fileHeader
	overwrittenRecords := []int64{}
	startingOffsets := [maxLevels]int64{}

	var rollbackErr error

KEYS_LOOP:
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
		c.setDirty(newRecordOffset, rec)
		dirtyOffsets = append(dirtyOffsets, newRecordOffset)
		for i := maxLevels - 1; i > level; i-- {
			offset, err := c.findLastLessThanOrEqual(key, startingOffsets[i], i, true, true)
			if err != nil {
				rollbackErr = err
				break KEYS_LOOP
			}
			if offset > 0 {
				startingOffsets[i] = offset
				if i > 0 {
					startingOffsets[i-1] = offset
				}
			}
		}

		for ; level >= 0; level-- {
			offset, err := c.findLastLessThanOrEqual(key, startingOffsets[level], level, true, true)
			if err != nil {
				rollbackErr = err
				break KEYS_LOOP
			}
			if offset == 0 {
				// Insert at head
				atomic.StoreInt64(&rec.Next[level], c.fileHeader.Next[level])
				atomic.StoreInt64(&c.fileHeader.Next[level], newRecordOffset)
			} else {
				// Have a previous record
				prevRec := &record{}
				if prev := c.getDirty(offset); prev != nil {
					prevRec = prev
				} else {
					readRec, err := c.readRecord(offset, true)
					if err != nil {
						rollbackErr = err
						break KEYS_LOOP
					}
					readRec.lock.RLock()
					*prevRec = *readRec
					readRec.lock.RUnlock()
				}
				atomic.StoreInt64(&rec.Next[level], prevRec.Next[level])
				atomic.StoreInt64(&prevRec.Next[level], newRecordOffset)
				c.setDirty(prevRec.Offset, prevRec)
				dirtyOffsets = append(dirtyOffsets, prevRec.Offset)
				walEntry.Push(newWALRecord(prevRec.Offset, prevRec.recordHeader.bytes()))

				if prevRec.Key == key && prevRec.Deleted == 0 {
					if !wb.allowOverwrite {
						rollbackErr = RollbackError{
							DuplicateKey:  true,
							ConflictedKey: key,
						}
						goto ROLLBACK
					}
					overwrittenRecords = append(overwrittenRecords, prevRec.Offset)
				}

				if level > 0 {
					startingOffsets[level-1] = prevRec.Offset
				}
			}

			startingOffsets[level] = newRecordOffset

			err = writeRecord(rec, appendBuf)
			if err != nil {
				rollbackErr = err
				break KEYS_LOOP
			}
		}
	}

	_, err = io.Copy(c.f, appendBuf)
	if err != nil {
		rollbackErr = fmt.Errorf("lm2: appending records failed (%s)", err)
		goto ROLLBACK
	}

	// Write sentinel record.
	currentOffset, err = c.writeSentinel()
	if err != nil {
		rollbackErr = err
		goto ROLLBACK
	}

	// fsync data file.
	err = c.f.Sync()
	if err != nil {
		rollbackErr = err
		goto ROLLBACK
	}

	c.dirtyLock.Lock()
	for _, dirtyRec := range c.dirty {
		walEntry.Push(newWALRecord(dirtyRec.Offset, dirtyRec.recordHeader.bytes()))
	}
	c.dirtyLock.Unlock()

	for key := range wb.deletes {
		offset := int64(0)
		for level := maxLevels - 1; level >= 0; level-- {
			offset, err = c.findLastLessThanOrEqual(key, offset, level, true, true)
			if err != nil {
				rollbackErr = err
				goto ROLLBACK
			}
		}
		if offset == 0 {
			continue
		}
		rec := &record{}
		if dirtyRec := c.getDirty(offset); dirtyRec != nil {
			rec = dirtyRec
		} else {
			readRec, err := c.readRecord(offset, true)
			if err != nil {
				rollbackErr = err
				goto ROLLBACK
			}
			readRec.lock.RLock()
			*rec = *readRec
			readRec.lock.RUnlock()
		}
		if rec.Key != key {
			continue
		}
		rec.Deleted = currentOffset
		c.setDirty(rec.Offset, rec)
		dirtyOffsets = append(dirtyOffsets, rec.Offset)
		walEntry.Push(newWALRecord(rec.Offset, rec.recordHeader.bytes()))
	}

	for _, offset := range overwrittenRecords {
		rec := &record{}
		if dirtyRec := c.getDirty(offset); dirtyRec != nil {
			rec = dirtyRec
		} else {
			readRec, err := c.readRecord(offset, true)
			if err != nil {
				rollbackErr = err
				goto ROLLBACK
			}
			readRec.lock.RLock()
			*rec = *readRec
			readRec.lock.RUnlock()
		}
		atomic.StoreInt64(&rec.Deleted, currentOffset)
		c.setDirty(rec.Offset, rec)
		dirtyOffsets = append(dirtyOffsets, rec.Offset)
		walEntry.Push(newWALRecord(rec.Offset, rec.recordHeader.bytes()))
	}

	c.LastCommit = currentOffset
	walEntry.Push(newWALRecord(0, c.fileHeader.bytes()))
	_, err = c.wal.Append(walEntry)
	if err != nil {
		rollbackErr = err
		goto ROLLBACK
	}

ROLLBACK:
	if rollbackErr != nil {
		// Do a rollback
		c.wal.Truncate()
		c.fileHeader.LastCommit = previousFileHeader.LastCommit
		for i, v := range previousFileHeader.Next {
			atomic.StoreInt64(&c.fileHeader.Next[i], v)
		}
		c.f.Truncate(c.LastCommit)

		c.cache.flushOffsets(dirtyOffsets)
		c.cache.lock.Lock()
		c.cache.cache = map[int64]*record{}
		c.cache.lock.Unlock()

		if IsRollbackError(rollbackErr) {
			return 0, rollbackErr
		}
		return 0, RollbackError{
			Err: rollbackErr,
		}
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

	c.cache.flushOffsets(dirtyOffsets)

	return c.LastCommit, nil
}
