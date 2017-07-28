package lm2

// WriteBatch represents a set of modifications.
type WriteBatch struct {
	sets    map[string]string
	deletes map[string]struct{}
}

// NewWriteBatch returns a new WriteBatch.
func NewWriteBatch() *WriteBatch {
	return &WriteBatch{
		sets:    map[string]string{},
		deletes: map[string]struct{}{},
	}
}

// Set adds key => value to the WriteBatch.
// Note: If a key is passed to Delete and Set,
// then the Set will be ignored.
func (wb *WriteBatch) Set(key, value string) {
	wb.sets[key] = value
}

// Delete marks a key for deletion.
func (wb *WriteBatch) Delete(key string) {
	wb.deletes[key] = struct{}{}
}

func (wb *WriteBatch) cleanup() {
	for key := range wb.deletes {
		delete(wb.sets, key)
	}
}
