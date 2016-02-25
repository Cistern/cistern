package source

import (
	"errors"
	"sync"

	"github.com/Cistern/cistern/message"
	"github.com/Cistern/cistern/state/metrics"
)

var ErrAddressAlreadyRegistered = errors.New("source: address already registered")

type Registry struct {
	sync.Mutex
	sources                map[string]*Source
	outboundGlobalMessages chan *message.Message
}

func NewRegistry(outbound chan *message.Message) *Registry {
	r := &Registry{
		Mutex:                  sync.Mutex{},
		sources:                map[string]*Source{},
		outboundGlobalMessages: outbound,
	}
	return r
}

func (r *Registry) RegisterSource(hostname, address string) (*Source, error) {
	if _, present := r.sources[address]; present {
		return nil, ErrAddressAlreadyRegistered
	}
	d := &Source{
		hostname:         hostname,
		address:          address,
		classes:          map[string]message.Class{},
		metrics:          metrics.NewMetricRegistry(),
		internalMessages: message.NewMessageChannel(),
		globalMessages:   r.outboundGlobalMessages,
	}
	go d.processMessages()
	r.sources[address] = d
	return d, nil
}

func (r *Registry) Lookup(address string) *Source {
	return r.sources[address]
}
