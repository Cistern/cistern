// Package net provides a Service, which handles all of Cistern's
// protocol and network integration.
package net

import (
	"log"
	"sync"

	appflowProto "github.com/Preetam/appflow"
	sflowProto "github.com/Preetam/sflow"

	"internal/net/appflow"
	"internal/net/sflow"
	"internal/source"
)

type Config struct {
	SFlowAddr   string `json:"sflowAddr"`
	AppFlowAddr string `json:"appflowAddr"`
}

var DefaultConfig = Config{
	SFlowAddr:   ":6343",
	AppFlowAddr: ":6344",
}

type Service struct {
	lock                         sync.Mutex
	sourceRegistry               *source.Registry
	sflowDatagrams               chan *sflowProto.Datagram
	sourceSFlowDatagramInbound   map[*source.Source]chan *sflowProto.Datagram
	appflowDatagrams             chan *appflowProto.HTTPFlowData
	sourceAppFlowDatagramInbound map[*source.Source]chan *appflowProto.HTTPFlowData
}

func NewService(conf Config, sourceRegistry *source.Registry) (*Service, error) {
	// TODO: use config
	s := &Service{
		lock:                         sync.Mutex{},
		sourceRegistry:               sourceRegistry,
		sflowDatagrams:               make(chan *sflowProto.Datagram, 1),
		sourceSFlowDatagramInbound:   map[*source.Source]chan *sflowProto.Datagram{},
		appflowDatagrams:             make(chan *appflowProto.HTTPFlowData, 1),
		sourceAppFlowDatagramInbound: map[*source.Source]chan *appflowProto.HTTPFlowData{},
	}
	_, err := sflow.NewDecoder(conf.SFlowAddr, s.sflowDatagrams)
	if err != nil {
		return nil, err
	}
	log.Println("listening for sFlow datagrams on", conf.SFlowAddr)
	_, err = appflow.NewDecoder(conf.AppFlowAddr, s.appflowDatagrams)
	if err != nil {
		return nil, err
	}
	log.Println("listening for AppFlow datagrams on", conf.AppFlowAddr)
	go s.dispatchSFlowDatagrams()
	go s.dispatchAppFlowDatagrams()
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
			s.sourceSFlowDatagramInbound[dev] = c
		}
		select {
		case s.sourceSFlowDatagramInbound[dev] <- dgram:
		default:
			// Drop.
		}
		dev.Unlock()
	}
}

func (s *Service) dispatchAppFlowDatagrams() {
	for _ = range s.appflowDatagrams {
		// TODO
	}
}
