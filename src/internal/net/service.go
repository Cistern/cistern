// Package net provides a Service, which handles all of Cistern's
// protocol and network integration.
package net

import (
	"log"
	"sync"

	sflowProto "github.com/Preetam/sflow"

	"internal/net/sflow"
	"internal/source"
)

type Config struct {
	SFlowAddr string `json:"sflowAddr"`
}

var DefaultConfig = Config{
	SFlowAddr: ":6343",
}

type Service struct {
	lock                  sync.Mutex
	sourceRegistry        *source.Registry
	sflowDatagrams        chan *sflowProto.Datagram
	sourceDatagramInbound map[*source.Source]chan *sflowProto.Datagram
}

func NewService(conf Config, sourceRegistry *source.Registry) (*Service, error) {
	// TODO: use config
	s := &Service{
		lock:                  sync.Mutex{},
		sourceRegistry:        sourceRegistry,
		sflowDatagrams:        make(chan *sflowProto.Datagram, 1),
		sourceDatagramInbound: map[*source.Source]chan *sflowProto.Datagram{},
	}
	_, err := sflow.NewDecoder(conf.SFlowAddr, s.sflowDatagrams)
	if err != nil {
		return nil, err
	}
	go s.dispatchSFlowDatagrams()
	return s, nil
}

func (s *Service) dispatchSFlowDatagrams() {
	for dgram := range s.sflowDatagrams {
		s.sourceRegistry.Lock()
		dev := s.sourceRegistry.Lookup(dgram.IpAddress)
		if dev == nil {
			var err error
			log.Println(dgram.IpAddress, "is unknown. Registering new source.")
			dev, err = s.sourceRegistry.RegisterSource("", dgram.IpAddress)
			if err != nil {
				log.Fatal(err)
			}
		}
		s.sourceRegistry.Unlock()
		dev.Lock()
		if !dev.HasClass("sflow") {
			log.Println(dev, "needs class \"sflow\".")
			c := make(chan *sflowProto.Datagram, 1)
			dev.RegisterClass(source.NewCommSFlowClass(dgram.IpAddress, c, dev.Messages()))
			s.sourceDatagramInbound[dev] = c
		}
		select {
		case s.sourceDatagramInbound[dev] <- dgram:
		default:
			// Drop.
		}
		dev.Unlock()
	}
}
