package metrics

type MessageContent map[string]struct {
	Type  MetricType
	Value interface{}
}
