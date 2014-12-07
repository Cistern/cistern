package series

import (
	"github.com/PreetamJinka/cistern/state/series/bolt"
)

type Engine struct {
	db *bolt.DB
}

func NewEngine(filename string) (*Engine, error) {
	db, err := bolt.Open(filename, 0666, nil)
	if err != nil {
		return nil, err
	}

	return &Engine{
		db: db,
	}, nil
}
