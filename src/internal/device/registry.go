package device

import (
	"errors"
	"net"
	"sync"

	"internal/device/class"
	"internal/message"
)

var ErrAddressAlreadyRegistered = errors.New("device: address already registered")

type mapIP [16]byte

type Registry struct {
	sync.Mutex
	devices  map[mapIP]*Device
	messages chan *message.Message
}

func NewRegistry() *Registry {
	r := &Registry{
		Mutex:    sync.Mutex{},
		devices:  map[mapIP]*Device{},
		messages: make(chan *message.Message, 1),
	}
	go r.processMessages()
	return r
}

func (r *Registry) processMessages() {
	for m := range r.messages {
		if !m.Global {
			panic("registry received non-global message")
		}
		// TODO
	}
}

func (r *Registry) RegisterDevice(hostname string, address net.IP) (*Device, error) {
	key := toMapIP(address)
	if _, present := r.devices[key]; present {
		return nil, ErrAddressAlreadyRegistered
	}
	d := &Device{
		hostname:         hostname,
		address:          address,
		classes:          map[string]class.Class{},
		internalMessages: make(chan *message.Message, 1),
		globalMessages:   r.messages,
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
