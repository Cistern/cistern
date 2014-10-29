package main

import (
	"flag"
	"log"

	"github.com/PreetamJinka/udpchan"

	"github.com/PreetamJinka/cistern/decode"
	"github.com/PreetamJinka/cistern/pipeline"
	"github.com/PreetamJinka/cistern/state/metrics"
)

var (
	sflowListenAddr = ":6343"
)

func main() {
	log.Printf("Cistern version %s starting", version)

	flag.StringVar(&sflowListenAddr, "sflow-listen-addr", sflowListenAddr, "listen address for sFlow datagrams")
	flag.Parse()

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
	processingPipeline.Add(pipeline.NewGenericIfaceProcessor(hostRegistry))
	processingPipeline.Add(pipeline.NewRawPacketProcessor(hostRegistry))

	pipelineMessages := make(chan pipeline.Message, 16)
	// TODO: refactor this part out
	go func() {
		for datagram := range sflowDecoder.Outbound() {
			source := datagram.Header.IpAddress.String()

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

	// make sure we don't exit
	<-make(chan struct{})
}
