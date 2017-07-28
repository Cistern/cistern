package lm2

import "sync/atomic"

// Stats holds collection statistics.
type Stats struct {
	RecordsWritten uint64
	RecordsRead    uint64
	CacheHits      uint64
	CacheMisses    uint64
}

func (s *Stats) incRecordsWritten(count uint64) {
	atomic.AddUint64(&s.RecordsWritten, count)
}

func (s *Stats) incRecordsRead(count uint64) {
	atomic.AddUint64(&s.RecordsRead, count)
}

func (s *Stats) incCacheHits(count uint64) {
	atomic.AddUint64(&s.CacheHits, count)
}

func (s *Stats) incCacheMisses(count uint64) {
	atomic.AddUint64(&s.CacheMisses, count)
}

func (s *Stats) clone() Stats {
	return Stats{
		RecordsWritten: atomic.LoadUint64(&s.RecordsWritten),
		RecordsRead:    atomic.LoadUint64(&s.RecordsRead),
		CacheHits:      atomic.LoadUint64(&s.CacheHits),
		CacheMisses:    atomic.LoadUint64(&s.CacheMisses),
	}
}
