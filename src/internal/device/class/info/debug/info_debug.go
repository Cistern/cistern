package debug

import (
	"log"

	"internal/message"
)

const ClassName = "debug"

type Class struct{}

func NewClass() *Class {
	return &Class{}
}

func (c *Class) Name() string {
	return ClassName
}

func (c *Class) Category() string {
	return "info"
}

func (c *Class) Process(m *message.Message) {
	log.Println(m)
}
