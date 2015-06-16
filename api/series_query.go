package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Preetam/catena"
	"github.com/VividCortex/siesta"
)

// A querySeries is an ordered set of points
// for a source and metric over a range
// of time.
type querySeries struct {
	// First timestamp
	Start int64 `json:"start"`

	// Last timestamp
	End int64 `json:"end"`

	Source string `json:"source"`
	Metric string `json:"metric"`

	Points []catena.Point `json:"points"`
}

// A queryDesc is a description of a
// query. It specifies a source, metric,
// start, and end timestamps.
type queryDesc struct {
	Source string `json:"source"`
	Metric string `json:"metric"`
	Start  int64  `json:"start"`
	End    int64  `json:"end"`
}

// A queryResponse is returned after querying
// the DB with a QueryDesc.
type queryResponse struct {
	Series []querySeries `json:"series"`
}

func (s *APIServer) querySeriesRoute() func(siesta.Context, http.ResponseWriter, *http.Request) {
	return func(c siesta.Context, w http.ResponseWriter, r *http.Request) {
		var params siesta.Params
		pointWidth := params.Int64("pointWidth", 1, "Number of points to average together")
		err := params.Parse(r.Form)
		if err != nil {
			c.Set(errorKey, err.Error())
			return
		}

		var descs []queryDesc

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

		resp := queryResponse{}

		for _, desc := range descs {
			log.Println(desc)
			i, err := s.seriesEngine.NewIterator(desc.Source, desc.Metric)
			if err != nil {
				log.Println(err)
				continue
			}

			err = i.Seek(desc.Start)
			if err != nil {
				log.Println(err)
				continue
			}

			s := querySeries{
				Source: desc.Source,
				Metric: desc.Metric,
				Start:  i.Point().Timestamp,
				End:    i.Point().Timestamp,
			}

			pointsSeen := 0

			currentInterval := i.Point().Timestamp / *pointWidth
			currentPoint := catena.Point{
				Timestamp: currentInterval * *pointWidth,
			}

			for {
				p := i.Point()
				if p.Timestamp > desc.End {
					break
				}

				if p.Timestamp / *pointWidth != currentInterval {
					currentPoint.Value /= float64(pointsSeen)
					s.Points = append(s.Points, currentPoint)
					currentInterval = i.Point().Timestamp / *pointWidth
					currentPoint = catena.Point{
						Timestamp: currentInterval * *pointWidth,
						Value:     p.Value,
					}
					pointsSeen = 1
					continue
				}

				currentPoint.Value += p.Value
				pointsSeen++

				err := i.Next()
				if err != nil {
					log.Println(err)
					break
				}
			}

			if pointsSeen > 0 {
				currentPoint.Value /= float64(pointsSeen)
				s.Points = append(s.Points, currentPoint)
			}
			i.Close()

			resp.Series = append(resp.Series, s)
		}

		c.Set(responseKey, resp)
	}
}
