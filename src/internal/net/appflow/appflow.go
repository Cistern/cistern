package appflow

import (
	"github.com/Preetam/appflow"
	"github.com/Preetam/udpchan"
)

// Decoder decodes AppFlow datagrams received over UDP.
type Decoder struct {
	listenAddr string
	inbound    <-chan []byte
	outbound   chan<- *appflow.HTTPFlowData
}

func NewDecoder(listenAddr string, outbound chan *appflow.HTTPFlowData) (*Decoder, error) {
	inbound, err := udpchan.Listen(listenAddr, nil)
	if err != nil {
		return nil, err
	}
	d := &Decoder{
		listenAddr: listenAddr,
		outbound:   outbound,
		inbound:    inbound,
	}
	go d.run()
	return d, nil
}

func (d *Decoder) run() {
	for buf := range d.inbound {
		flowData, err := appflow.Decode(buf)
		if err == nil {
			select {
			case d.outbound <- flowData:
			default:
				// Drop.
			}
		}
	}
}
