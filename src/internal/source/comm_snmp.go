package source

import (
	"github.com/Preetam/snmp"
	"internal/message"
)

const CommSNMPClassName = "snmp"

type CommSNMPClass struct {
	session *snmp.Session
	inbound chan *message.Message
}

func (c *CommSNMPClass) Name() string {
	return CommSNMPClassName
}

func (c *CommSNMPClass) Category() string {
	return "comm"
}

func (c *CommSNMPClass) InboundMessages() chan *message.Message {
	return c.inbound
}
