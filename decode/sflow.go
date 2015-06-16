package decode

import (
	"github.com/Preetam/sflow"

	"bytes"
	"log"
)

type SflowDecoder struct {
	inbound  <-chan []byte
	outbound chan sflow.Datagram
}

func NewSflowDecoder(inbound <-chan []byte, bufferLength ...int) *SflowDecoder {
	bufLen := 0

	if len(bufferLength) > 0 {
		bufLen = bufferLength[0]
	}

	return &SflowDecoder{
		inbound:  inbound,
		outbound: make(chan sflow.Datagram, bufLen),
	}
}

func (d *SflowDecoder) Outbound() chan sflow.Datagram {
	return d.outbound
}

func (d *SflowDecoder) Run() {
	decoder := sflow.NewDecoder(nil)
	go func() {
		for buf := range d.inbound {
			r := bytes.NewReader(buf)

			decoder.Use(r)

			dgram, err := decoder.Decode()
			if err == nil {
				d.outbound <- *dgram
			} else {
				log.Println(err)
			}
		}
	}()
}
