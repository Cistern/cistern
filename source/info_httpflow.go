package source

import (
	"log"

	"github.com/Cistern/appflow"

	"github.com/Cistern/cistern/message"
)

const InfoHTTPFlowClassName = "http-flow"

type InfoHTTPFlowClass struct {
	sourceAddress string
	outbound      chan *message.Message
}

func NewInfoHTTPFlowClass(sourceAddress string, outbound chan *message.Message) *InfoHTTPFlowClass {
	c := &InfoHTTPFlowClass{
		sourceAddress: sourceAddress,
		outbound:      outbound,
	}
	return c
}

func (c *InfoHTTPFlowClass) Name() string {
	return InfoHTTPFlowClassName
}

func (c *InfoHTTPFlowClass) Category() string {
	return "info"
}

func (c *InfoHTTPFlowClass) OutboundMessages() chan *message.Message {
	return c.outbound
}

func (c *InfoHTTPFlowClass) Process(m *message.Message) {
	record := m.Content.(*appflow.HTTPFlowData)
	log.Println("HTTP Flow:", record.Proto, record.Method, record.Host, record.URL, record.Header)
}
