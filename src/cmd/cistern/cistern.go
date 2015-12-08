// Cistern is a flow collector.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"internal/clock"
	"internal/config"
	"internal/message"
	"internal/net"
	"internal/source"
	"internal/state/series"
)

var (
	sflowListenAddr = ":6343"
	apiListenAddr   = ":8080"
	configFile      = "/opt/cistern/config.json"
	seriesDataDir   = ""
	commitSHA       = ""
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	// Flags
	flag.StringVar(&sflowListenAddr, "sflow-listen-addr",
		sflowListenAddr, "listen address for sFlow datagrams")
	flag.StringVar(&apiListenAddr, "api-listen-addr",
		apiListenAddr, "listen address for HTTP API server")
	flag.StringVar(&configFile, "config",
		configFile, "configuration file")
	flag.StringVar(&seriesDataDir, "series-data-dir",
		seriesDataDir, "directory to store time series data (disabled by default)")
	showVersion := flag.Bool("version", false, "Show version")
	showLicense := flag.Bool("license", false, "Show software licenses")
	showConfig := flag.Bool("show-config", false, "Show loaded config file")
	flag.Parse()

	if *showVersion {
		shaString := ""
		if commitSHA != "" {
			shaString = " (" + commitSHA + ")"
		}
		fmt.Printf("Cistern version %s%s", version, shaString)
		os.Exit(0)
	}

	if *showLicense {
		fmt.Println(license)
		os.Exit(0)
	}

	log.Printf("Cistern version %s starting", version)
	clock.Run()

	log.Printf("  Attempting to load configuration file at %s", configFile)
	conf, err := config.Load(configFile)
	if err != nil {
		log.Printf("✗ Could not load configuration: `%v`", err)
	}

	// Log the loaded config
	confBytes, err := json.MarshalIndent(conf, "  ", "  ")
	if err != nil {
		log.Printf("✗ Could not log config: `%v`", err)
	} else {
		if *showConfig {
			log.Println("\n  " + string(confBytes))
		}
		log.Println("✓ Successfully loaded configuration")
	}

	if seriesDataDir != "" {
		log.Printf("  Starting series engine using %s", seriesDataDir)
		engine, err := series.NewEngine(seriesDataDir)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("✓ Successfully started series engine")

		var _ = engine
	}

	globalMessages := message.NewMessageChannel()
	registry := source.NewRegistry(globalMessages)
	_, err = net.NewService(net.DefaultConfig, registry)
	if err != nil {
		log.Fatalf("✗ failed to start network service: %v", err)
	}
	log.Println("✓ Successfully started network service")

	// make sure we don't exit
	<-make(chan struct{})
}
