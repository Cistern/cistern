package main

import (
	"github.com/PreetamJinka/sflow-go"
	"github.com/PreetamJinka/udpchan"

	"flag"
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	listenAddr := flag.String("listen", ":6343", "address of the sFlow datagram collector")
	flag.Parse()

	// Start listening over UDP.
	c, err := udpchan.Listen(*listenAddr, nil)
	if err != nil {
		log.Fatalln(err)
	}

	registry := NewHostRegistry()

	p := &Pipeline{}

	p.Add(NewHostProcessor(registry))
	p.Add(NewGenericIfaceProcessor(registry))

	// This is just for fun!
	go printHostStats(registry)

	// Create a channel for pipeline messages.
	messages := make(chan Message)
	// Start the pipeline processes.
	go p.Run(messages)

	go http.ListenAndServe(":8080", ServeHostCpuStats(registry))

	// buf is a UDP payload.
	for buf := range c {
		// Decode it.
		dgram := sflow.Decode(buf)
		ip := dgram.Header.IpAddress

		for _, sample := range dgram.Samples {
			switch sample.SampleType() {

			// We're only interested in counters (for now).
			case sflow.TypeCounterSample, sflow.TypeExpandedCounterSample:
				for _, rec := range sample.GetRecords() {

					// Send the records through the pipeline
					// using the source IP as a host key.
					messages <- Message{
						Source: ip.String(),
						Record: rec,
					}
				}
			}
		}
	}
}

// printHostStats prints relative CPU utilization statistics
// for each host once a second.
func printHostStats(registry *HostRegistry) {
	for _ = range time.Tick(time.Second) {
		hosts := registry.GetHosts()

	HOST_LOOP:
		for _, host := range hosts {

			// These are the metrics we're interested in.
			cpuMetrics := []string{
				"cpu.user",
				"cpu.sys",
				"cpu.nice",
				"cpu.wio",
				"cpu.intr",
				"cpu.softintr",
				"cpu.idle",
			}

			// Get values. Some, or all, of these could be NaN.
			metrics, err := registry.Query(host, cpuMetrics...)
			if err != nil {
				continue
			}

			var totalTime float32

			for _, metric := range metrics {
				// NaN != NaN according to the IEEE standard.
				if metric != metric {
					continue HOST_LOOP
				}
				totalTime += metric
			}

			// We want percentages.
			totalTime /= 100

			// Print them all!
			fmt.Printf("[%s] %.02f%%us %.02f%%sys %.02f%%ni %.02f%%io %.02f%%in %.02f%%si %.02f%%id\n",
				host, metrics[0]/totalTime,
				metrics[1]/totalTime,
				metrics[2]/totalTime,
				metrics[3]/totalTime,
				metrics[4]/totalTime,
				metrics[5]/totalTime,
				metrics[6]/totalTime,
			)
		}
	}
}
