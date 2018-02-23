package lm2

import (
	"math/rand"
	"sync"
)

type recordCache struct {
	cache        map[int64]*record
	maxKeyRecord *record
	size         int
	preventPurge bool
	lock         sync.RWMutex
}

func newCache(size int) *recordCache {
	return &recordCache{
		cache:        map[int64]*record{},
		maxKeyRecord: nil,
		size:         size,
	}
}

func (rc *recordCache) findLastLessThan(key string) int64 {
	rc.lock.RLock()
	defer rc.lock.RUnlock()

	if rc.maxKeyRecord != nil {
		if rc.maxKeyRecord.Key < key {
			return rc.maxKeyRecord.Offset
		}
	}
	max := ""
	maxOffset := int64(0)

	for offset, record := range rc.cache {
		if record.Key >= key {
			continue
		}
		if record.Key > max {
			max = record.Key
			maxOffset = offset
		}
	}
	return maxOffset
}

func (rc *recordCache) push(rec *record) {
	rc.lock.RLock()

	if rc.maxKeyRecord == nil || rc.maxKeyRecord.Key < rec.Key {
		rc.lock.RUnlock()

		rc.lock.Lock()
		if rc.maxKeyRecord == nil || rc.maxKeyRecord.Key < rec.Key {
			rc.maxKeyRecord = rec
		}
		rc.lock.Unlock()

		return
	}

	if len(rc.cache) == rc.size && rand.Float32() >= cacheProb {
		rc.lock.RUnlock()
		return
	}

	rc.lock.RUnlock()
	rc.lock.Lock()

	rc.cache[rec.Offset] = rec
	if !rc.preventPurge {
		rc.purge()
	}

	rc.lock.Unlock()
}

func (rc *recordCache) purge() {
	purged := 0
	for len(rc.cache) > rc.size {
		deletedKey := int64(0)
		for k := range rc.cache {
			if k == rc.maxKeyRecord.Offset {
				continue
			}
			deletedKey = k
			break
		}
		delete(rc.cache, deletedKey)
		purged++
	}
}

func (rc *recordCache) flushOffsets(offsets []int64) {
	rc.lock.Lock()
	for _, offset := range offsets {
		delete(rc.cache, offset)
	}
	rc.lock.Unlock()
}
