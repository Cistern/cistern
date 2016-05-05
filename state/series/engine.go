package series

import (
	"log"
	"unsafe"

	"github.com/Cistern/catena"
	"github.com/Cistern/cistern/message"
)

const SeriesEngineClassName = "series-engine"

type Engine struct {
	*catena.DB
}

func (*Engine) Name() string {
	return SeriesEngineClassName
}

func (*Engine) Category() string {
	return "" // unused
}

func NewEngine(baseDir string) (*Engine, error) {
	db, err := catena.OpenDB(baseDir, 3600, 24*90)
	if err != nil {
		log.Println("Could not open DB at", baseDir)
		log.Println("Creating new DB at", baseDir)
		db, err = catena.NewDB(baseDir, 3600, 24*90)
		if err != nil {
			return nil, err
		}
	}
	engine := &Engine{
		DB: db,
	}
	return engine, nil
}

func (engine *Engine) Process(m *message.Message) {
	observations, ok := m.Content.([]Observation)
	if !ok {
		// Drop
		return
	}
	engine.writeObservations(observations)
}

func (engine *Engine) writeObservations(obs []Observation) {
	log.Printf("[Series engine] Writing %d observations", len(obs))
	engine.DB.InsertRows(*(*[]catena.Row)(unsafe.Pointer(&obs)))
}
