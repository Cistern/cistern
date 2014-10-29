package pipeline

type BlackholeProcessor struct {
	inbound chan Message
}

func (b *BlackholeProcessor) SetInbound(inbound chan Message) {
	b.inbound = inbound
}

func (b *BlackholeProcessor) Process() {
	for _ = range b.inbound {
		// poof
	}
}

func (b *BlackholeProcessor) Outbound() chan Message {
	return nil
}
