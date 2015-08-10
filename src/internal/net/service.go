// Package net provides a Service, which handles all of Cistern's
// protocol and network integration.
package net

import (
	"log"
	"sync"

	sflowProto "github.com/Preetam/sflow"

	"internal/device"
	commSflow "internal/device/class/comm/sflow"
	"internal/net/sflow"
)

type Config struct {
	SFlowAddr string `json:"sflowAddr"`
}

var DefaultConfig = Config{
	SFlowAddr: ":6343",
}

type Service struct {
	lock                  sync.Mutex
	deviceRegistry        *device.Registry
	sflowDatagrams        chan *sflowProto.Datagram
	deviceDatagramInbound map[*device.Device]chan *sflowProto.Datagram
}

func NewService(conf Config, deviceRegistry *device.Registry) (*Service, error) {
	// TODO: use config
	s := &Service{
		lock:                  sync.Mutex{},
		deviceRegistry:        deviceRegistry,
		sflowDatagrams:        make(chan *sflowProto.Datagram),
		deviceDatagramInbound: map[*device.Device]chan *sflowProto.Datagram{},
	}
	_, err := sflow.NewDecoder(conf.SFlowAddr, s.sflowDatagrams)
	if err != nil {
		return nil, err
	}
	go s.dispatchSflowDatagrams()
	return s, nil
}

func (s *Service) dispatchSflowDatagrams() {
	for dgram := range s.sflowDatagrams {
		log.Printf("received a datagram from %v", dgram.IpAddress)
		s.deviceRegistry.Lock()
		dev := s.deviceRegistry.Lookup(dgram.IpAddress)
		if dev == nil {
			var err error
			log.Println(dgram.IpAddress, "is unknown. Registering new device.")
			dev, err = s.deviceRegistry.RegisterDevice("", dgram.IpAddress)
			if err != nil {
				log.Fatal(err)
			}
		}
		s.deviceRegistry.Unlock()
		dev.Lock()
		if !dev.HasClass("sflow") {
			log.Println(dev, "needs class \"sflow\".")
			c := make(chan *sflowProto.Datagram, 1)
			dev.RegisterClass(commSflow.NewClass(dgram.IpAddress, c, dev.Messages()))
			s.deviceDatagramInbound[dev] = c
		}
		select {
		case s.deviceDatagramInbound[dev] <- dgram:
		default:
			// Drop.
		}
		dev.Unlock()
	}
}
