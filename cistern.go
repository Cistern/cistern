package main

import (
	"flag"
	"log"
	"os"

	"github.com/PreetamJinka/cistern/decode"
	"github.com/PreetamJinka/udpchan"
	"github.com/VividCortex/trace"
)

var (
	sflowListenAddr = ":6343"

	traceEnabled = false
)

func main() {
	log.Printf("Cistern version %s starting", version)

	flag.StringVar(&sflowListenAddr, "sflow-listen-addr", sflowListenAddr, "listen address for sFlow datagrams")
	flag.BoolVar(&traceEnabled, "trace", traceEnabled, "enable trace output")
	flag.Parse()

	if traceEnabled {
		trace.SetWriter(os.Stderr)
		trace.Enable()
		log.Println("tracing is enabled")
	} else {
		trace.Disable()
		log.Println("tracing is disabled")
	}

	// start listening
	c, listenErr := udpchan.Listen(sflowListenAddr, nil)
	if listenErr != nil {
		log.Fatalf("failed to start listening: [%s]", listenErr)
	}

	log.Printf("listening for sFlow datagrams on %s", sflowListenAddr)

	// start a decoder
	sflowDecoder := decode.NewSflowDecoder(c, 16)
	go sflowDecoder.Run()

	// make sure we don't exit
	<-make(chan struct{})
}
