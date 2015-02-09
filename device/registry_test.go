package device

import (
	"net"
	"testing"
)

func TestRegistry(t *testing.T) {
	reg := NewRegistry()

	t.Log(reg.Lookup(net.ParseIP("127.0.0.1")))

	dev := NewDevice(net.ParseIP("127.0.0.1"))

	reg.devices[dev.ip.String()] = dev
	t.Log(reg.Lookup(net.ParseIP("127.0.0.1")))
}
