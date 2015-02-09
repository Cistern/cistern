package series

import (
	"github.com/PreetamJinka/catena"
)

type Engine struct {
	*catena.DB
}

func NewEngine(baseDir string) (*Engine, error) {
	db, err := catena.OpenDB(baseDir)

	if err != nil {
		db, err = catena.NewDB(baseDir)
		if err != nil {
			return nil, err
		}
	}

	return &Engine{
		db,
	}, nil
}
