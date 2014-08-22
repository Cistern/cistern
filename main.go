package main

import (
	"github.com/PreetamJinka/sflow-go"
	"github.com/PreetamJinka/udpchan"

	"flag"
	"fmt"
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
			hosts := registry.GetHosts()

		HOST_LOOP:
			for _, host := range hosts {
				cpuMetrics := []string{
					"cpu.user",
					"cpu.sys",
					"cpu.nice",
					"cpu.wio",
					"cpu.intr",
					"cpu.softintr",
					"cpu.idle",
				}
				metrics, err := registry.Query(host, cpuMetrics...)
				if err != nil {
					continue
				}

				var totalTime float32

				for _, metric := range metrics {
					if metric != metric {
						continue HOST_LOOP
					}
					totalTime += metric
				}

				totalTime /= 100

				fmt.Printf("[%s] %.02f%%us %.02f%%sys %.02f%%ni %.02f%%io %.02f%%in %.02f%%si %.02f%%id\n",
					host, metrics[0]/totalTime,
					metrics[1]/totalTime,
					metrics[2]/totalTime,
					metrics[3]/totalTime,
					metrics[4]/totalTime,
					metrics[5]/totalTime,
					metrics[6]/totalTime)
			}
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
