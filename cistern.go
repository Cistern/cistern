package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"runtime"

	"github.com/Preetam/udpchan"

	"github.com/Preetam/cistern/api"
	"github.com/Preetam/cistern/config"
	"github.com/Preetam/cistern/decode"
	"github.com/Preetam/cistern/device"
	"github.com/Preetam/cistern/state/series"
)

var (
	sflowListenAddr = ":6343"
	apiListenAddr   = ":8080"
	configFile      = "/opt/cistern/config.json"
)

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	// Flags
	flag.StringVar(&sflowListenAddr, "sflow-listen-addr", sflowListenAddr, "listen address for sFlow datagrams")
	flag.StringVar(&apiListenAddr, "api-listen-addr", apiListenAddr, "listen address for HTTP API server")
	flag.StringVar(&configFile, "config", configFile, "configuration file")
	showVersion := flag.Bool("version", false, "Show version")
	showLicense := flag.Bool("license", false, "Show software licenses")
	showConfig := flag.Bool("show-config", false, "Show loaded config file")
	flag.Parse()

	log.SetFlags(log.Lshortfile | log.Lmicroseconds)

	if *showVersion {
		fmt.Println("Cistern version", version)
		os.Exit(0)
	}

	if *showLicense {
		fmt.Println(license)
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
		if *showConfig {
			log.Println("\n  " + string(confBytes))
		}
	}

	engine, err := series.NewEngine("/tmp/cistern/catena")
	if err != nil {
		log.Fatal(err)
	}

	registry, err := device.NewRegistry(engine.Inbound)
	if err != nil {
		log.Fatal(err)
	}

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

			err = registry.SetDeviceSNMP(ip, addr, dev.SNMP.User, dev.SNMP.AuthPassphrase, dev.SNMP.PrivPassphrase)
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

	go func() {
		for datagram := range sflowDecoder.Outbound() {
			source := datagram.IpAddress

			registryDev := registry.LookupOrAdd(source)
			registryDev.Inbound <- datagram
		}
	}()

	apiServer := api.NewAPIServer(apiListenAddr, registry, engine)
	apiServer.Run()
	log.Printf("API server started on %s", apiListenAddr)

	// make sure we don't exit
	<-make(chan struct{})
}
