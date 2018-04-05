package query

// Executor is a query executor.
type Executor struct{}

func NewExecutor(t Table) *Executor {
	return &Executor{}
}

func (e *Executor) Execute(query Desc) error {
	// TODO
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
