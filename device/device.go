package device

import (
	"log"
	"net"
	"sync"

	"github.com/PreetamJinka/cistern/net/snmp"
)

type deviceType int

const (
	TypeUnknown deviceType = 0

	TypeNetwork deviceType = 1 << (iota - 1)
	TypeLinux
)

var (
	descOid     = snmp.MustParseOID(".1.3.6.1.2.1.1.1.0")
	hostnameOid = snmp.MustParseOID(".1.3.6.1.2.1.1.5.0")
)

// A Device is an entity that sends flows or
// makes information available via SNMP.
type Device struct {
	hostname    string
	desc        string
	ip          net.IP
	snmpSession *snmp.Session
}

// NewDevice returns a new Device with the given IP.
func NewDevice(address net.IP) *Device {
	return &Device{
		ip: address,
	}
}

func (d *Device) Discover() {
	wg := sync.WaitGroup{}

	wg.Add(2)
	go func() {
		defer wg.Done()

		// Discover hostname
		getRes, err := d.snmpSession.Get(hostnameOid)
		if err != nil {
			log.Printf("[SNMP %v] Could not get hostname: %v", d.ip, err)
			return
		}

		if vbinds := getRes.Varbinds(); len(vbinds) > 0 {
			hostnameStr, err := vbinds[0].GetStringValue()
			if err != nil {
				log.Printf("[SNMP %v] Invalid GetResponse for hostname: %v", d.ip, err)
				return
			}

			d.hostname = hostnameStr

			log.Printf("[SNMP %v] Discovered hostname %s", d.ip, hostnameStr)
		}
	}()

	go func() {
		defer wg.Done()

		// Discover description
		getRes, err := d.snmpSession.Get(descOid)
		if err != nil {
			log.Printf("[SNMP %v] Could not get description: %v", d.ip, err)
			return
		}

		if vbinds := getRes.Varbinds(); len(vbinds) > 0 {
			descStr, err := vbinds[0].GetStringValue()
			if err != nil {
				log.Printf("[SNMP %v] Invalid GetResponse for description: %v", d.ip, err)
				return
			}

			d.desc = descStr

			log.Printf("[SNMP %v] Discovered desc %s", d.ip, descStr)
		}
	}()

	wg.Wait()
}
