package device

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"

	"internal/device/class/info/debug"
	"internal/device/class/info/host_counters"
	"internal/message"
)

var (
	ErrClassNameRegistered = errors.New("device: class name already registered")
)

type Device struct {
	sync.Mutex

	hostname string
	address  net.IP

	classes          map[string]message.Class
	internalMessages chan *message.Message
	globalMessages   chan<- *message.Message
}

func (d *Device) RegisterClass(c message.Class) error {
	if _, present := d.classes[c.Name()]; present {
		return ErrClassNameRegistered
	}
	d.classes[c.Name()] = c
	return nil
}

func (d *Device) HasClass(classname string) bool {
	_, present := d.classes[classname]
	return present
}

func (d *Device) Messages() chan *message.Message {
	return d.internalMessages
}

// processMessages delivers messages from internal device classes to
// other internal device classes, or escalates them to the global
// channel.
func (d *Device) processMessages() {
	for m := range d.internalMessages {
		if m.Global {
			// m is intended for a global class.
			select {
			case d.globalMessages <- m:
			default:
				// Drop.
			}
			continue
		}
		if !d.HasClass(m.Class) {
			log.Printf("  %v does not have class \"%s\" registered", d, m.Class)
			switch m.Class {
			case host_counters.ClassName:
				d.RegisterClass(host_counters.NewClass(d.address, d.internalMessages))
			case debug.ClassName:
				d.RegisterClass(debug.NewClass())
			default:
				continue
			}
		}
		c := d.classes[m.Class]
		if processor, ok := c.(message.Processor); ok {
			processor.Process(m)
		}
	}
}

func (d *Device) String() string {
	if d.hostname == "" {
		return fmt.Sprintf("Device{%v}", d.address)
	}
	return fmt.Sprintf("Device{%s - %v}", d.hostname, d.address)
}
