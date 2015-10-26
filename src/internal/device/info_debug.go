package device

import (
	"log"

	"internal/message"
)

const InfoDebugClassName = "debug"

type InfoDebugClass struct{}

func NewInfoDebugClass() *InfoDebugClass {
	return &InfoDebugClass{}
}

func (c *InfoDebugClass) Name() string {
	return InfoDebugClassName
}

func (c *InfoDebugClass) Category() string {
	return "info"
}

func (c *InfoDebugClass) Process(m *message.Message) {
	log.Println(m)
}
