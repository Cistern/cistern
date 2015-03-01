package flows

import (
	"bytes"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/PreetamJinka/cistern/state/metrics"
)

const (
	FlowProtocolTCP = "TCP"
	FlowProtocolUDP = "UDP"

	alpha float32 = 1
)

type Flow struct {
	Source          net.IP `json:"source"`
	Destination     net.IP `json:"destination"`
	SourcePort      int    `json:"sourcePort"`
	DestinationPort int    `json:"destinationPort"`

	Protocol string `json:"protocol"`

	bytesDerivative   metrics.MetricState
	packetsDerivative metrics.MetricState

	bytes   uint64
	packets uint64

	BytesPerSecond   float32 `json:"bytesPerSecond"`
	PacketsPerSecond float32 `json:"packetsPerSecond"`

	lastUpdated time.Time
}

type flowKey struct {
	protocol string
	addr1    [16]byte
	port1    int
	addr2    [16]byte
	port2    int
}

func toFlowKey(protocol string, source, destination net.IP, sourcePort, destPort int) flowKey {
	source = source.To16()
	destination = destination.To16()

	key := flowKey{
		protocol: protocol,
	}

	if bytes.Compare([]byte(source), []byte(destination)) < 0 {
		copy(key.addr1[:], source)
		copy(key.addr2[:], destination)
		key.port1 = sourcePort
		key.port2 = destPort
		return key
	}

	copy(key.addr1[:], destination)
	copy(key.addr2[:], source)
	key.port1 = destPort
	key.port2 = sourcePort
	return key
}

type TopTalkers struct {
	flows map[flowKey]Flow

	lock sync.Mutex
}

func NewTopTalkers(window time.Duration) *TopTalkers {
	toptalkers := &TopTalkers{
		flows: map[flowKey]Flow{},
	}

	go func() {
		for _ = range time.Tick(window) {
			toptalkers.compact(window)
		}
	}()

	return toptalkers
}

func (t *TopTalkers) compact(window time.Duration) {
	t.lock.Lock()
	defer t.lock.Unlock()

	timeout := time.Now().Add(-window)

	for key, flow := range t.flows {
		if flow.lastUpdated.Before(timeout) {
			delete(t.flows, key)
		}
	}
}

func (t *TopTalkers) Update(protocol string,
	source, destination net.IP, sourcePort, destPort int, bytes int) {
	t.lock.Lock()
	defer t.lock.Unlock()

	key := toFlowKey(protocol, source, destination, sourcePort, destPort)
	var flow Flow

	var present bool
	if flow, present = t.flows[key]; !present {
		flow.Source = source
		flow.Destination = destination
		flow.SourcePort = sourcePort
		flow.DestinationPort = destPort
		flow.bytesDerivative = metrics.DerivativeState{}
		flow.packetsDerivative = metrics.DerivativeState{}
		flow.Protocol = protocol
	}

	seenBytes := uint64(bytes) + flow.bytes
	flow.bytesDerivative = flow.bytesDerivative.Update(seenBytes)
	flow.BytesPerSecond = flow.bytesDerivative.Value()
	flow.bytes = seenBytes

	seenPackets := flow.packets + 1
	flow.packetsDerivative = flow.packetsDerivative.Update(seenPackets)
	flow.PacketsPerSecond = flow.packetsDerivative.Value()
	flow.packets = seenPackets

	flow.lastUpdated = time.Now()
	t.flows[key] = flow
}

func (t *TopTalkers) ByBytes() []Flow {
	t.lock.Lock()
	defer t.lock.Unlock()

	flows := make([]Flow, 0, len(t.flows))
	for _, flow := range t.flows {
		flows = append(flows, flow)
	}

	sort.Sort(sort.Reverse(byBytes(flows)))
	return flows
}

func (t *TopTalkers) ByPackets() []Flow {
	t.lock.Lock()
	defer t.lock.Unlock()

	flows := make([]Flow, 0, len(t.flows))
	for _, flow := range t.flows {
		flows = append(flows, flow)
	}

	sort.Sort(sort.Reverse(byPackets(flows)))
	return flows
}

type byBytes []Flow

func (f byBytes) Len() int {
	return len(f)
}

func (f byBytes) Less(i, j int) bool {
	return f[i].BytesPerSecond < f[j].BytesPerSecond
}

func (f byBytes) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

type byPackets []Flow

func (f byPackets) Len() int {
	return len(f)
}

func (f byPackets) Less(i, j int) bool {
	return f[i].PacketsPerSecond < f[j].PacketsPerSecond
}

func (f byPackets) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}
