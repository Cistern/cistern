package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/PreetamJinka/catena"
	"github.com/VividCortex/siesta"
)

func (s *APIServer) querySeriesRoute() func(siesta.Context, http.ResponseWriter, *http.Request) {
	return func(c siesta.Context, w http.ResponseWriter, r *http.Request) {
		var params siesta.Params
		downsample := params.Int64("downsample", 0, "A downsample value of averages N points at a time")
		err := params.Parse(r.Form)
		if err != nil {
			c.Set(errorKey, err.Error())
			return
		}

		var descs []catena.QueryDesc

		dec := json.NewDecoder(r.Body)
		err = dec.Decode(&descs)
		if err != nil {
			c.Set(errorKey, err.Error())
			return
		}

		now := time.Now().Unix()
		for i, desc := range descs {
			if desc.Start <= 0 {
				desc.Start += now
			}

			if desc.End <= 0 {
				desc.End += now
			}

			descs[i] = desc
		}

		resp := s.seriesEngine.Query(descs)

		if *downsample <= 1 {
			c.Set(responseKey, resp)
			return
		}

		for i, series := range resp.Series {
			pointIndex := 0
			seenPoints := 1
			currentPartition := series.Points[0].Timestamp / *downsample
			for j, p := range series.Points {
				if j == 0 {
					continue
				}

				if p.Timestamp / *downsample == currentPartition {
					series.Points[pointIndex].Value += p.Value
					seenPoints++
				} else {
					currentPartition = p.Timestamp / *downsample
					series.Points[pointIndex].Value /= float64(seenPoints)
					pointIndex++
					seenPoints = 1
					series.Points[pointIndex] = p
				}

				if j == len(series.Points) {
					series.Points[pointIndex].Value /= float64(seenPoints)
				}
			}

			series.Points = series.Points[:pointIndex]
			resp.Series[i] = series
		}

		c.Set(responseKey, resp)
	}
}
