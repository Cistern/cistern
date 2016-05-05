package api

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/VividCortex/siesta"

	"github.com/Cistern/catena"
	"github.com/Cistern/cistern/source"
	"github.com/Cistern/cistern/state/series"
)

const (
	responseKey = "response"
	errorKey    = "error"
)

type API struct {
	addr           string
	sourceRegistry *source.Registry
	seriesEngine   *series.Engine
}

type APIResponse struct {
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

func NewAPI(addr string, seriesEngine *series.Engine) *API {
	return &API{
		addr:         addr,
		seriesEngine: seriesEngine,
	}
}

func (api *API) Run() {
	service := siesta.NewService("/")

	service.AddPre(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", r.Header.Get("Access-Control-Request-Headers"))
	})

	service.AddPost(func(c siesta.Context, w http.ResponseWriter, r *http.Request, q func()) {
		resp := c.Get(responseKey)
		err, _ := c.Get(errorKey).(string)

		if resp == nil && err == "" {
			return
		}

		enc := json.NewEncoder(w)
		enc.Encode(APIResponse{
			Data:  resp,
			Error: err,
		})
	})

	service.Route("GET", "/sources", "Lists sources", func(c siesta.Context, w http.ResponseWriter, r *http.Request) {
		var params siesta.Params
		start := params.Int64("start", -3600, "Start timestamp")
		end := params.Int64("start", 0, "End timestamp")
		err := params.Parse(r.Form)
		if err != nil {
			c.Set(errorKey, err.Error())
			return
		}

		now := time.Now().Unix()
		if *start < 0 {
			*start += now
		}
		if *end <= 0 {
			*end += now
		}
		sources := api.seriesEngine.Sources(*start, *end)
		c.Set(responseKey, sources)
	})

	service.Route("GET", "/sources/:source/metrics",
		"Lists metrics for a source",
		func(c siesta.Context, w http.ResponseWriter, r *http.Request) {
			var params siesta.Params
			source := params.String("source", "", "Source name")
			start := params.Int64("start", -3600, "Start timestamp")
			end := params.Int64("start", 0, "End timestamp")
			err := params.Parse(r.Form)
			if err != nil {
				c.Set(errorKey, err.Error())
				return
			}

			now := time.Now().Unix()
			if *start < 0 {
				*start += now
			}
			if *end <= 0 {
				*end += now
			}

			metrics := api.seriesEngine.DB.Metrics(*source, *start, *end)
			c.Set(responseKey, metrics)
		})

	service.Route("OPTIONS", "/series/query",
		"Accepts an OPTIONS request",
		func(w http.ResponseWriter, r *http.Request) {
			// Doesn't do anything
		})

	service.Route("POST", "/series/query",
		"Lists metrics for a source",
		api.querySeriesRoute())

	http.ListenAndServe(api.addr, service)
}

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

func (api *API) querySeriesRoute() func(siesta.Context, http.ResponseWriter, *http.Request) {
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
			i, err := api.seriesEngine.DB.NewIterator(desc.Source, desc.Metric)
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
