package series

import (
	"log"
	"time"
	"unsafe"

	"github.com/PreetamJinka/catena"
)

type Engine struct {
	*catena.DB
	Inbound chan Observation
}

func NewEngine(baseDir string) (*Engine, error) {
	db, err := catena.OpenDB(baseDir, 3600, 24*90)

	if err != nil {
		log.Println(err)

		db, err = catena.NewDB(baseDir, 3600, 24*90)
		if err != nil {
			return nil, err
		}
	}

	engine := &Engine{
		DB:      db,
		Inbound: make(chan Observation, 512),
	}

	go engine.handleInbound()

	return engine, nil
}

func (engine *Engine) handleInbound() {
	log.Println("[Series engine] Handling inbound observations")

	var wait = time.After(time.Second * 5)
	var buffer = make([]Observation, 0, 512)

	for {
		select {
		case <-wait:
			if len(buffer) > 0 {
				engine.writeObservations(buffer)
				buffer = buffer[:0]
			}

			wait = time.After(time.Second * 5)

		case obs := <-engine.Inbound:
			buffer = append(buffer, obs)
			if len(buffer) == 512 {
				engine.writeObservations(buffer)
				buffer = buffer[:0]
				wait = time.After(time.Second * 5)
			}
		}
	}
}

func (engine *Engine) writeObservations(obs []Observation) {
	log.Printf("[Series engine] Writing %d observations", len(obs))
	engine.DB.InsertRows(*(*[]catena.Row)(unsafe.Pointer(&obs)))
}
