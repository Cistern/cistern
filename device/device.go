package device

import (
	"log"
	"net"
	"sync"
	"time"

	"github.com/PreetamJinka/sflow"
	"github.com/PreetamJinka/snmp"

	"github.com/PreetamJinka/cistern/state/flows"
	"github.com/PreetamJinka/cistern/state/metrics"
	"github.com/PreetamJinka/cistern/state/series"
)

type deviceType int

const (
	TypeUnknown deviceType = 0

	TypeNetwork deviceType = 1 << (iota - 1)
	TypeLinux
	TypeBSD
)

var (
	descOid     = snmp.MustParseOID(".1.3.6.1.2.1.1.1.0")
	hostnameOid = snmp.MustParseOID(".1.3.6.1.2.1.1.5.0")
)

// A Device is an entity that sends flows or
// makes information available via SNMP.
type Device struct {
	hostname   string
	desc       string
	ip         net.IP
	deviceType deviceType

	snmpSession    *snmp.Session
	metricRegistry *metrics.MetricRegistry
	topTalkers     *flows.TopTalkers

	Inbound  chan sflow.Datagram
	outbound chan series.Observation
}

// NewDevice returns a new Device with the given IP.
func NewDevice(address net.IP, outboundObservations chan series.Observation) *Device {
	dev := &Device{
		ip: address,

		metricRegistry: metrics.NewMetricRegistry(),

		Inbound:  make(chan sflow.Datagram),
		outbound: outboundObservations,
	}

	go dev.handleFlows()

	return dev
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

func (d *Device) Metrics() []metrics.MetricDefinition {
	return d.metricRegistry.Metrics()
}

func (d *Device) IP() net.IP {
	return d.ip
}

func (d *Device) Hostname() string {
	return d.hostname
}

func (d *Device) TopTalkers() *flows.TopTalkers {
	return d.topTalkers
}

func (d *Device) handleFlows() {

	log.Printf("[Device %v] Handling flows", d.ip)

	for dgram := range d.Inbound {
		for _, sample := range dgram.Samples {
			for _, record := range sample.GetRecords() {
				d.processFlowRecord(record)
			}
		}
	}
}

func (d *Device) processFlowRecord(r sflow.Record) {
	switch r.(type) {
	case sflow.HostCpuCounters:
		d.processHostCPUCounters(r.(sflow.HostCpuCounters))
	case sflow.HostMemoryCounters:
		d.processHostMemoryCounters(r.(sflow.HostMemoryCounters))
	case sflow.HostDiskCounters:
		d.processHostDiskCounters(r.(sflow.HostDiskCounters))
	case sflow.HostNetCounters:
		d.processHostNetCounters(r.(sflow.HostNetCounters))
	case sflow.GenericInterfaceCounters:
		d.processGenericInterfaceCounters(r.(sflow.GenericInterfaceCounters))
	case sflow.RawPacketFlow:
		if d.topTalkers == nil {
			d.topTalkers = flows.NewTopTalkers(time.Second * 30)
		}
		d.processRawPacketFlow(r.(sflow.RawPacketFlow))
	}
}

func (d *Device) updateAndEmit(metric string, metricType metrics.MetricType, v interface{}) {
	value := d.metricRegistry.Update(metric, metricType, v)
	d.outbound <- series.Observation{
		Source:    d.ip.String(),
		Metric:    metric,
		Timestamp: time.Now().Unix(),
		Value:     float64(value),
	}
}
