package lm2

import "sync/atomic"

// Cursor represents a snapshot cursor.
type Cursor struct {
	collection *Collection
	current    *record
	first      bool
	snapshot   int64
	err        error
}

// NewCursor returns a new cursor with a snapshot view of the
// current collection state.
func (c *Collection) NewCursor() (*Cursor, error) {
	if atomic.LoadUint32(&c.internalState) != 0 {
		return nil, ErrInternal
	}

	c.metaLock.RLock()
	defer c.metaLock.RUnlock()
	if c.Next[0] == 0 {
		return &Cursor{
			collection: c,
			current:    nil,
			first:      false,
			snapshot:   c.LastCommit,
		}, nil
	}

	head, err := c.readRecord(c.Next[0])
	if err != nil {
		return nil, err
	}
	cur := &Cursor{
		collection: c,
		current:    head,
		first:      true,
		snapshot:   c.LastCommit,
	}

	var rec *record
	cur.current.lock.RLock()
	for (cur.current.Deleted != 0 && cur.current.Deleted <= cur.snapshot) ||
		(cur.current.Offset >= cur.snapshot) {
		rec, err = cur.collection.readRecord(atomic.LoadInt64(&cur.current.Next[0]))
		if err != nil {
			cur.current.lock.RUnlock()
			cur.current = nil
			cur.err = err
			return cur, nil
		}
		cur.current.lock.RUnlock()
		cur.current = rec
		cur.current.lock.RLock()
	}
	cur.current.lock.RUnlock()

	return cur, nil
}

// Valid returns true if the cursor's Key() and Value()
// methods can be called. It returns false if the cursor
// isn't at a valid record position.
func (c *Cursor) Valid() bool {
	return c.current != nil
}

// Next moves the cursor to the next record. It returns true
// if it lands on a valid record.
func (c *Cursor) Next() bool {
	if atomic.LoadUint32(&c.collection.internalState) != 0 {
		c.current = nil
		return false
	}

	if !c.Valid() {
		return false
	}

	if c.first {
		c.first = false
		return true
	}

	c.current.lock.RLock()
	rec, err := c.collection.readRecord(atomic.LoadInt64(&c.current.Next[0]))
	if err != nil {
		c.current.lock.RUnlock()
		if atomic.LoadInt64(&c.current.Next[0]) != 0 {
			c.err = err
		}
		c.current = nil
		return false
	}
	c.current.lock.RUnlock()
	c.current = rec

	c.current.lock.RLock()
	for (c.current.Deleted != 0 && c.current.Deleted <= c.snapshot) ||
		(c.current.Offset >= c.snapshot) {
		rec, err = c.collection.readRecord(atomic.LoadInt64(&c.current.Next[0]))
		if err != nil {
			c.current.lock.RUnlock()
			if atomic.LoadInt64(&c.current.Next[0]) != 0 {
				c.err = err
			}
			c.current = nil
			return false
		}
		c.current.lock.RUnlock()
		c.current = rec
		c.current.lock.RLock()
	}
	c.current.lock.RUnlock()

	return true
}

// Key returns the key of the current record. It returns an empty
// string if the cursor is not valid.
func (c *Cursor) Key() string {
	if c.Valid() {
		return c.current.Key
	}
	return ""
}

// Value returns the value of the current record. It returns an
// empty string if the cursor is not valid.
func (c *Cursor) Value() string {
	if c.Valid() {
		return c.current.Value
	}
	return ""
}

// Seek positions the cursor at the last key less than
// or equal to the provided key.
func (c *Cursor) Seek(key string) {
	if atomic.LoadUint32(&c.collection.internalState) != 0 {
		return
	}

	var err error
	offset := int64(0)
	for level := maxLevels - 1; level >= 0; level-- {
		offset, err = c.collection.findLastLessThanOrEqual(key, offset, level, false)
		if err != nil {
			c.err = err
			return
		}
	}
	if offset == 0 {
		c.collection.metaLock.RLock()
		offset = c.collection.Next[0]
		c.collection.metaLock.RUnlock()

		if offset == 0 {
			c.current = nil
			return
		}
	}
	rec, err := c.collection.readRecord(offset)
	if err != nil {
		c.err = err
		return
	}

	c.current = rec
	c.first = true
	for rec != nil {
		rec.lock.RLock()
		if rec.Key >= key {
			if (rec.Deleted > 0 && rec.Deleted <= c.snapshot) || (rec.Offset >= c.snapshot) {
				oldRec := rec
				rec, err = c.collection.nextRecord(rec, 0)
				if err != nil {
					if atomic.LoadInt64(&c.current.Next[0]) != 0 {
						c.err = err
					}
					c.current = nil
					return
				}
				oldRec.lock.RUnlock()
				c.current = rec
				continue
			}
			rec.lock.RUnlock()
			break
		}
		if (rec.Deleted > 0 && rec.Deleted <= c.snapshot) || (rec.Offset >= c.snapshot) {
			oldRec := rec
			rec, err = c.collection.nextRecord(rec, 0)
			if err != nil {
				if atomic.LoadInt64(&c.current.Next[0]) != 0 {
					c.err = err
				}
				c.current = nil
				return
			}
			oldRec.lock.RUnlock()
			continue
		}
		if rec.Key < key {
			c.current = rec
		}
		oldRec := rec
		rec, err = c.collection.nextRecord(rec, 0)
		if err != nil {
			if atomic.LoadInt64(&c.current.Next[0]) != 0 {
				c.err = err
			}
			c.current = nil
			return
		}
		oldRec.lock.RUnlock()
	}
}

// Err returns the error encountered during iteration, if any.
func (c *Cursor) Err() error {
	return c.err
}
