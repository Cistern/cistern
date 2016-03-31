package series

import "github.com/Cistern/catena"

type Observation struct {
	Source    string
	Metric    string
	Timestamp int64
	Value     float64
}

func (o Observation) toCatenaRow() catena.Row {
	return catena.Row{
		Source: o.Source,
		Metric: o.Metric,
		Point: catena.Point{
			Timestamp: o.Timestamp,
			Value:     o.Value,
		},
	}
}
