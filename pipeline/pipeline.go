package pipeline

import (
	"log"

	"github.com/PreetamJinka/cistern/net/sflow"
)

type Pipeline struct {
	processors []PipelineProcessor
}

func (p *Pipeline) Add(proc PipelineProcessor) {
	p.processors = append(p.processors, proc)
}

func (p *Pipeline) Run(inbound chan Message) {
	log.Println("starting pipeline")
	for _, proc := range p.processors {
		proc.SetInbound(inbound)
		inbound = proc.Outbound()
		go proc.Process()
	}

	go (&BlackholeProcessor{inbound: inbound}).Process()
}

type Message struct {
	Source string
	Record sflow.Record
}

type PipelineProcessor interface {
	Process()
	SetInbound(chan Message)
	Outbound() chan Message
}
