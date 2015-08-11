package device

import (
	"errors"
	"net"
	"sync"

	"internal/message"
	"internal/state/metrics"
)

var ErrAddressAlreadyRegistered = errors.New("device: address already registered")

type mapIP [16]byte

type Registry struct {
	sync.Mutex
	devices                map[mapIP]*Device
	outboundGlobalMessages chan *message.Message
}

func NewRegistry(outbound chan *message.Message) *Registry {
	r := &Registry{
		Mutex:                  sync.Mutex{},
		devices:                map[mapIP]*Device{},
		outboundGlobalMessages: outbound,
	}
	return r
}

func (r *Registry) RegisterDevice(hostname string, address net.IP) (*Device, error) {
	key := toMapIP(address)
	if _, present := r.devices[key]; present {
		return nil, ErrAddressAlreadyRegistered
	}
	d := &Device{
		hostname:         hostname,
		address:          address,
		classes:          map[string]message.Class{},
		metrics:          metrics.NewMetricRegistry(),
		internalMessages: message.NewMessageChannel(),
		globalMessages:   r.outboundGlobalMessages,
	}
	go d.processMessages()
	r.devices[key] = d
	return d, nil
}

func (r *Registry) Lookup(address net.IP) *Device {
	return r.devices[toMapIP(address)]
}

func toMapIP(ip net.IP) mapIP {
	mIP := mapIP{}
	copy(mIP[:], ip.To16())
	return mIP
}
