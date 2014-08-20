package main

import (
	"github.com/PreetamJinka/sflow-go"
	"github.com/PreetamJinka/udpchan"

	"flag"
	"log"
	"time"
)

func main() {
	listenAddr := flag.String("listen", ":6343", "address of the sFlow datagram collector")
	flag.Parse()

	c, err := udpchan.Listen(*listenAddr, nil)
	if err != nil {
		log.Fatalln(err)
	}

	registry := NewHostRegistry()

	p := &Pipeline{}

	p.Add(NewHostProcessor(registry))
	p.Add(NewGenericIfaceProcessor(registry))

	go func() {
		for _ = range time.Tick(time.Second) {
			log.Print(registry)
		}
	}()

	messages := make(chan Message)
	go p.Run(messages)

	for buf := range c {
		dgram := sflow.Decode(buf)
		ip := dgram.Header.IpAddress

		for _, sample := range dgram.Samples {
			switch sample.SampleType() {
			case sflow.TypeCounterSample, sflow.TypeExpandedCounterSample:
				for _, rec := range sample.GetRecords() {
					messages <- Message{
						Source: ip.String(),
						Record: rec,
					}
				}
			}
		}
	}
}
