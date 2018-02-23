package lm2

// WriteBatch represents a set of modifications.
type WriteBatch struct {
	sets           map[string]string
	deletes        map[string]struct{}
	allowOverwrite bool
}

// NewWriteBatch returns a new WriteBatch.
func NewWriteBatch() *WriteBatch {
	return &WriteBatch{
		sets:           map[string]string{},
		deletes:        map[string]struct{}{},
		allowOverwrite: true,
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

// AllowOverwrite determines whether keys will be overwritten.
// If allow is false and an existing key is being
// set, updates will be rolled back.
func (wb *WriteBatch) AllowOverwrite(allow bool) {
	wb.allowOverwrite = allow
}

func (wb *WriteBatch) cleanup() {
	for key := range wb.deletes {
		delete(wb.sets, key)
	}
}
