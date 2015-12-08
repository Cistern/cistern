package source

import (
	"errors"
	"net"
	"sync"

	"internal/message"
	"internal/state/metrics"
)

var ErrAddressAlreadyRegistered = errors.New("source: address already registered")

type mapIP [16]byte

type Registry struct {
	sync.Mutex
	sources                map[mapIP]*Source
	outboundGlobalMessages chan *message.Message
}

func NewRegistry(outbound chan *message.Message) *Registry {
	r := &Registry{
		Mutex:                  sync.Mutex{},
		sources:                map[mapIP]*Source{},
		outboundGlobalMessages: outbound,
	}
	return r
}

func (r *Registry) RegisterSource(hostname string, address net.IP) (*Source, error) {
	key := toMapIP(address)
	if _, present := r.sources[key]; present {
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
	r.sources[key] = d
	return d, nil
}

func (r *Registry) Lookup(address net.IP) *Source {
	return r.sources[toMapIP(address)]
}

func toMapIP(ip net.IP) mapIP {
	mIP := mapIP{}
	copy(mIP[:], ip.To16())
	return mIP
}
