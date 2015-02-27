package device

import (
	"net"
	"sync"

	"github.com/PreetamJinka/snmp"
)

// A Registry is a device registry.
type Registry struct {

	// devices is a map from IP address string to Device.
	devices map[string]*Device

	sessionManager *snmp.SessionManager

	sync.Mutex
}

// NewRegistry creates a new device registry.
func NewRegistry() (*Registry, error) {
	sessionManager, err := snmp.NewSessionManager()
	if err != nil {
		return nil, err
	}

	return &Registry{
		devices:        map[string]*Device{},
		sessionManager: sessionManager,
	}, nil
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

func (r *Registry) SetDeviceSNMP(deviceAddress net.IP, snmpAddr, user, auth, priv string) error {
	d := r.LookupOrAdd(deviceAddress)

	sess, err := r.sessionManager.NewSession(snmpAddr, user, auth, priv)
	if err != nil {
		return err
	}

	err = sess.Discover()
	if err != nil {
		return err
	}

	d.snmpSession = sess
	return nil
}
