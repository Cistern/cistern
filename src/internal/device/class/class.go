package class

import (
	"internal/device/class/comm/sflow"
	"internal/device/class/info/host_counters"
	"internal/message"
)

type Class interface {
	Name() string
	Category() string
}

type Emitter interface {
	OutboundMessages() chan *message.Message
}

type Collector interface {
	InboundMessages() chan *message.Message
}

// *sflow.Class is a Class.
var _ Class = &sflow.Class{}

// *sflow.Class is an Emitter.
var _ Emitter = &sflow.Class{}

// *host_counters.Class is a Class.
var _ Class = &host_counters.Class{}

// *host_counters.Class is a Collector.
var _ Collector = &host_counters.Class{}

// *host_counters.Class is an Emitter.
var _ Emitter = &host_counters.Class{}
