package device

import (
	"net"
	"sync"
)

// A Registry is a device registry.
type Registry struct {

	// devices is a map from IP address string to Device.
	devices map[string]*Device

	sync.Mutex
}

// NewRegistry creates a new device registry.
func NewRegistry() *Registry {
	return &Registry{
		devices: map[string]*Device{},
	}
}

// Devices returns a slice of devices in the registry.
func (r *Registry) Devices() []*Device {
	r.Lock()
	defer r.Unlock()

	devices := []*Device{}

	for _, device := range r.devices {
		devices = append(devices, device)
	}

	return devices
}

// NumDevices returns the number of devices in the registry.
func (r *Registry) NumDevices() int {
	return len(r.devices)
}

func (r *Registry) Lookup(address net.IP) (*Device, bool) {
	dev, present := r.devices[address.String()]

	return dev, present
}

func (r *Registry) LookupOrAdd(address net.IP) *Device {
	r.Lock()
	defer r.Unlock()

	dev, present := r.Lookup(address)
	if present {
		return dev
	}

	dev = NewDevice(address)
	r.devices[address.String()] = dev
	return dev
}
