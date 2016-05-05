// Package net provides a Service, which handles all of Cistern's
// protocol and network integration.
package net

import (
	"log"
	"sync"

	appflowProto "github.com/Cistern/appflow"
	sflowProto "github.com/Cistern/sflow"

	apiPackage "github.com/Cistern/cistern/net/api"
	"github.com/Cistern/cistern/net/appflow"
	"github.com/Cistern/cistern/net/sflow"
	"github.com/Cistern/cistern/source"
	"github.com/Cistern/cistern/state/series"
)

type Config struct {
	SFlowAddr   string `json:"sflowAddr"`
	AppFlowAddr string `json:"appflowAddr"`
	APIAddr     string `json:"apiAddr"`
}

var DefaultConfig = Config{
	SFlowAddr:   ":6343",
	AppFlowAddr: ":6344",
	APIAddr:     ":6345",
}

type Service struct {
	lock                         sync.Mutex
	sourceRegistry               *source.Registry
	sflowDatagrams               chan *sflowProto.Datagram
	sourceSFlowDatagramInbound   map[*source.Source]chan *sflowProto.Datagram
	appflowDatagrams             chan *appflowProto.HTTPFlowData
	sourceAppFlowDatagramInbound map[*source.Source]chan *appflowProto.HTTPFlowData
}

func NewService(conf Config, sourceRegistry *source.Registry, seriesEngine *series.Engine) (*Service, error) {
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
	api := apiPackage.NewAPI(conf.APIAddr, seriesEngine)
	go s.dispatchSFlowDatagrams()
	go s.dispatchAppFlowDatagrams()
	go api.Run()
	return s, nil
}

func (s *Service) dispatchSFlowDatagrams() {
	for dgram := range s.sflowDatagrams {
		s.sourceRegistry.Lock()
		src := s.sourceRegistry.Lookup(dgram.IpAddress.String())
		if src == nil {
			var err error
			log.Println(dgram.IpAddress, "is unknown. Registering new source.")
			src, err = s.sourceRegistry.RegisterSource("", dgram.IpAddress.String())
			if err != nil {
				log.Fatal(err)
			}
		}
		s.sourceRegistry.Unlock()
		src.Lock()
		if !src.HasClass("sflow") {
			log.Println(src, "needs class \"sflow\".")
			c := make(chan *sflowProto.Datagram, 1)
			src.RegisterClass(source.NewCommSFlowClass(dgram.IpAddress, c, src.Messages()))
			s.sourceSFlowDatagramInbound[src] = c
		}
		select {
		case s.sourceSFlowDatagramInbound[src] <- dgram:
		default:
			// Drop.
		}
		src.Unlock()
	}
}

func (s *Service) dispatchAppFlowDatagrams() {
	for dgram := range s.appflowDatagrams {
		s.sourceRegistry.Lock()
		src := s.sourceRegistry.Lookup(dgram.Host)
		if src == nil {
			var err error
			log.Println(dgram.Host, "is unknown. Registering new source.")
			src, err = s.sourceRegistry.RegisterSource("", dgram.Host)
			if err != nil {
				log.Fatal(err)
			}
		}
		s.sourceRegistry.Unlock()
		src.Lock()
		if !src.HasClass("appflow") {
			log.Println(src, "needs class \"appflow\".")
			c := make(chan *appflowProto.HTTPFlowData, 1)
			src.RegisterClass(source.NewCommAppFlowClass(c, src.Messages()))
			s.sourceAppFlowDatagramInbound[src] = c
		}
		select {
		case s.sourceAppFlowDatagramInbound[src] <- dgram:
		default:
			// Drop.
		}
		src.Unlock()
	}
}
