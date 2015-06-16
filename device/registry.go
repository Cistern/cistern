package device

import (
	"net"
	"sync"

	"github.com/Preetam/snmp"

	"github.com/Preetam/cistern/state/series"
)

// A Registry is a device registry.
type Registry struct {

	// devices is a map from IP address (max 16 bytes) string to Device.
	devices map[[16]byte]*Device

	sessionManager *snmp.SessionManager

	outbound chan series.Observation

	sync.Mutex
}

// NewRegistry creates a new device registry.
func NewRegistry(outbound chan series.Observation) (*Registry, error) {
	sessionManager, err := snmp.NewSessionManager()
	if err != nil {
		return nil, err
	}

	return &Registry{
		devices:        map[[16]byte]*Device{},
		sessionManager: sessionManager,
		outbound:       outbound,
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
	byteAddr := [16]byte{}
	copy(byteAddr[:], address.To16())

	dev, present := r.devices[byteAddr]

	return dev, present
}

func (r *Registry) LookupOrAdd(address net.IP) *Device {
	r.Lock()
	defer r.Unlock()

	dev, present := r.Lookup(address)
	if present {
		return dev
	}

	dev = NewDevice(address, r.outbound)

	byteAddr := [16]byte{}
	copy(byteAddr[:], address.To16())

	r.devices[byteAddr] = dev
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
