package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/Cistern/cistern/internal/query"
	"github.com/Preetam/siesta"
)

func service() *siesta.Service {
	service := siesta.NewService("/api")
	service.Route("OPTIONS", "/collections/:collection/query", "preflight request", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", r.Header.Get("Access-Control-Request-Headers"))
	})
	service.Route("POST", "/collections/:collection/query", "query a collection", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")

		var params siesta.Params
		collectionName := params.String("collection", "", "collection name")
		queryString := params.String("query", "", "query string")
		start := params.Int64("start", 0, "Start Unix timestamp")
		end := params.Int64("end", 0, "End Unix timestamp")
		err := params.Parse(r.Form)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		collectionsLock.Lock()
		collection, present := Collections[*collectionName]
		collectionsLock.Unlock()

		if !present {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		queryDesc, err := query.Parse(*queryString)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		log.Println("Got query", *queryString)

		queryDesc.TimeRange.Start = time.Unix(*start, 0)
		queryDesc.TimeRange.End = time.Unix(*end, 0)

		if queryDesc.PointSize > 0 {
			// Round off timestamps
			queryDesc.TimeRange.Start = queryDesc.TimeRange.Start.Truncate(time.Duration(queryDesc.PointSize * 1000))
			queryDesc.TimeRange.End = queryDesc.TimeRange.End.Truncate(time.Duration(queryDesc.PointSize * 1000))
		}

		log.Println(queryDesc.TimeRange)

		for i, filter := range queryDesc.Filters {
			var v interface{}
			err := json.Unmarshal([]byte(filter.Value.(json.RawMessage)), &v)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				log.Println(err)
				return
			}
			filter.Value = v
			queryDesc.Filters[i] = filter
		}

		result, err := collection.Query(*queryDesc)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err)
		}

		json.NewEncoder(w).Encode(result)
	})

	service.Route("POST", "/collections/:collection/compact", "compacts a collection", func(w http.ResponseWriter, r *http.Request) {
		var params siesta.Params
		collectionName := params.String("collection", "", "collection name")
		err := params.Parse(r.Form)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		collectionsLock.Lock()
		collection, present := Collections[*collectionName]
		collectionsLock.Unlock()

		if !present {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		err = collection.Compact()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err)
		}
	})
	return service
}
