package main

import (
	"flag"
	"log"

	"github.com/PreetamJinka/udpchan"

	"github.com/PreetamJinka/cistern/api"
	"github.com/PreetamJinka/cistern/config"
	"github.com/PreetamJinka/cistern/decode"
	"github.com/PreetamJinka/cistern/net/snmp"
	"github.com/PreetamJinka/cistern/pipeline"
	"github.com/PreetamJinka/cistern/state/metrics"
	"github.com/PreetamJinka/cistern/state/series"
)

var (
	sflowListenAddr = ":6343"
	apiListenAddr   = ":8080"
	configFile      = "/opt/cistern/config.json"

	descOid     = snmp.MustParseOID(".1.3.6.1.2.1.1.1.0")
	hostnameOid = snmp.MustParseOID(".1.3.6.1.2.1.1.5.0")
)

func main() {
	flag.StringVar(&sflowListenAddr, "sflow-listen-addr", sflowListenAddr, "listen address for sFlow datagrams")
	flag.StringVar(&apiListenAddr, "api-listen-addr", apiListenAddr, "listen address for HTTP API server")
	flag.StringVar(&configFile, "config", configFile, "configuration file")
	flag.Parse()

	log.Printf("Cistern version %s starting", version)

	log.Printf("Loading configuration file at %s", configFile)

	conf, err := config.Load(configFile)
	if err != nil {
		log.Print(err)
	}

	for _, device := range conf.SNMPDevices {
		// TODO: refactor this part out
		go func(dev config.SNMPEntry) {
			session, err := snmp.NewSession(dev.Address, dev.User, dev.AuthPassphrase, dev.PrivPassphrase)
			if err != nil {
				log.Println(err)
				return
			}

			err = session.Discover()
			if err != nil {
				log.Printf("[SNMP] Discovery failed for %s", dev.Address)
				return
			}

			resp, err := session.Get(descOid)
			if err != nil {
				log.Printf("[SNMP] Get desc failed for %s", dev.Address)
				return
			}

			deviceDesc := ""

			if vbinds := resp.Varbinds(); len(vbinds) > 0 {
				deviceDesc, err = vbinds[0].GetStringValue()
				if err != nil {
					log.Printf("[SNMP] Did not get a string value for device description for %s", dev.Address)
					return
				}
			}

			resp, err = session.Get(hostnameOid)
			if err != nil {
				log.Printf("[SNMP] Get hostname failed for %s", dev.Address)
				return
			}

			deviceHostname := ""

			if vbinds := resp.Varbinds(); len(vbinds) > 0 {
				deviceHostname, err = vbinds[0].GetStringValue()
				if err != nil {
					log.Printf("[SNMP] Did not get a string value for device hostname for %s", dev.Address)
					return
				}
			}

			log.Printf("[SNMP] Discovery\n at %s [%s]:\n  %s", dev.Address, deviceHostname, deviceDesc)
		}(device)
	}

	// start listening
	c, listenErr := udpchan.Listen(sflowListenAddr, nil)
	if listenErr != nil {
		log.Fatalf("failed to start listening: [%s]", listenErr)
	}

	log.Printf("listening for sFlow datagrams on %s", sflowListenAddr)

	// start a decoder
	sflowDecoder := decode.NewSflowDecoder(c, 16)
	sflowDecoder.Run()

	hostRegistry := metrics.NewHostRegistry()

	processingPipeline := &pipeline.Pipeline{}
	processingPipeline.Add(pipeline.NewHostProcessor(hostRegistry))
	processingPipeline.Add(pipeline.NewGenericInterfaceCountersProcessor(hostRegistry))
	processingPipeline.Add(pipeline.NewRawPacketProcessor(hostRegistry))

	pipelineMessages := make(chan pipeline.Message, 16)
	// TODO: refactor this part out
	go func() {
		for datagram := range sflowDecoder.Outbound() {
			source := datagram.IpAddress.String()

			for _, sample := range datagram.Samples {
				for _, record := range sample.GetRecords() {
					pipelineMessages <- pipeline.Message{
						Source: source,
						Record: record,
					}
				}
			}
		}
	}()
	processingPipeline.Run(pipelineMessages)

	go LogDiagnostics(hostRegistry)

	engine, err := series.NewEngine("/tmp/cistern.db")
	if err != nil {
		log.Fatal(err)
	}

	api := api.NewApiServer(apiListenAddr, hostRegistry, engine)
	api.Run()
	log.Printf("started API server listening on %s", apiListenAddr)

	go hostRegistry.RunSnapshotter(engine)

	// make sure we don't exit
	<-make(chan struct{})
}
