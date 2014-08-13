package main

import (
	"github.com/PreetamJinka/sflow-go"
	"github.com/PreetamJinka/udpchan"

	"fmt"
	"log"
	"time"
)

func main() {
	c, err := udpchan.Listen(":6343", nil)
	if err != nil {
		log.Fatalln(err)
	}

	registry := NewHostRegistry()

	go func() {
		for _ = range time.Tick(time.Second) {
			fmt.Println(registry)
		}
	}()

	p := &Pipeline{}

	p.Add(NewHostProcessor(registry))
	p.Add(NewGenericIfaceProcessor(registry))

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
