package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/PreetamJinka/udpchan"

	"github.com/PreetamJinka/cistern/api"
	"github.com/PreetamJinka/cistern/config"
	"github.com/PreetamJinka/cistern/decode"
	"github.com/PreetamJinka/cistern/device"
	"github.com/PreetamJinka/cistern/pipeline"
	"github.com/PreetamJinka/cistern/state/metrics"
	"github.com/PreetamJinka/cistern/state/series"
)

var (
	sflowListenAddr = ":6343"
	apiListenAddr   = ":8080"
	configFile      = "/opt/cistern/config.json"
)

func main() {

	// Flags
	flag.StringVar(&sflowListenAddr, "sflow-listen-addr", sflowListenAddr, "listen address for sFlow datagrams")
	flag.StringVar(&apiListenAddr, "api-listen-addr", apiListenAddr, "listen address for HTTP API server")
	flag.StringVar(&configFile, "config", configFile, "configuration file")
	showVersion := flag.Bool("version", false, "Show version")
	flag.Parse()

	log.SetFlags(log.Lshortfile | log.Lmicroseconds)

	if *showVersion {
		fmt.Println("Cistern version", version)
		os.Exit(0)
	}

	log.Printf("Cistern version %s starting", version)

	log.Printf("Attempting to load configuration file at %s", configFile)

	conf, err := config.Load(configFile)
	if err != nil {
		log.Printf("Could not load configuration: %v", err)
	}

	// Log the loaded config
	confBytes, err := json.MarshalIndent(conf, "  ", "  ")
	if err != nil {
		log.Println("Could not log config:", err)
	} else {
		log.Println("\n  " + string(confBytes))
	}

	registry := device.NewRegistry()
	for _, dev := range conf.Devices {

		ip := net.ParseIP(dev.IP)
		// Add a device to the registry
		registryDev := registry.LookupOrAdd(ip)

		if dev.SNMP != nil {
			// We have an SNMP config
			addr := ip.String()
			if ip.To4() == nil {
				// IPv6 addresses need to be surrounded
				// with `[` and `]`.
				addr = "[" + addr + "]"
			}

			port := 161

			if dev.SNMP.Port != 0 {
				port = dev.SNMP.Port
			}

			addr = fmt.Sprintf("%s:%d", addr, port)

			err = registryDev.SetSNMP(addr, dev.SNMP.User, dev.SNMP.AuthPassphrase, dev.SNMP.PrivPassphrase)
			if err == nil {
				log.Println("Successfully created SNMP session with", addr)
				log.Println("Starting device discovery")

				registryDev.Discover()
			} else {
				log.Printf("SNMP session creation failed for %v: %v", addr, err)
			}
		}
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
