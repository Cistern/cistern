package query

// Executor is a query executor.
type Executor struct {
	table Table
}

func NewExecutor(table Table) *Executor {
	return &Executor{
		table: table,
	}
}

func (e *Executor) Execute(query Desc) error {
	// Get a cursor
	var cur Cursor
	var err error
	if len(query.Columns) == 0 && len(query.GroupBy) == 0 {
		// No aggregations. Use a plain cursor.
		cur, err = e.table.NewCursor()
		if err != nil {
			return err
		}
	} else {
		fieldsMap := map[string]struct{}{}
		for _, col := range query.Columns {
			fieldsMap[col.Name] = struct{}{}
		}
		for _, col := range query.GroupBy {
			fieldsMap[col.Name] = struct{}{}
		}
		fields := make([]string, 0, len(fieldsMap))
		for field := range fieldsMap {
			fields = append(fields, field)
		}
		cur, err = e.table.NewCursorForFields(fields)
		if err != nil {
			return err
		}
	}

	// TODO
	_ = cur

	return nil
}

type Table interface {
	NewCursor() (Cursor, error)
	NewCursorForFields(fields []string) (Cursor, error)
}

type Cursor interface {
	GetRow() (Row, error)
	Next() bool
	Err() error
}

type Seeker interface {
	Seek(Row) error
}

type Row interface {
	Fields() []string
	Get(field string) (interface{}, error)
}
