package sflow

import (
	"bytes"

	"github.com/Cistern/sflow"
	"github.com/Cistern/udpchan"
)

// Decoder decodes sFlow datagrams received over UDP.
type Decoder struct {
	listenAddr string
	inbound    <-chan []byte
	outbound   chan<- *sflow.Datagram
}

func NewDecoder(listenAddr string, outbound chan *sflow.Datagram) (*Decoder, error) {
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
	datagramDecoder := sflow.NewDecoder(nil)
	for buf := range d.inbound {
		r := bytes.NewReader(buf)
		datagramDecoder.Use(r)
		dgram, err := datagramDecoder.Decode()
		if err == nil {
			select {
			case d.outbound <- dgram:
			default:
				// Drop.
			}
		}
	}
}
