package main

import (
	"compress/gzip"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"path/filepath"
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

		if !present {
			collection, err = OpenEventCollection(filepath.Join(DataDir, *collectionName+".lm2"))
			if err != nil {
				collectionsLock.Unlock()
				if err == ErrDoesNotExist {
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			Collections[*collectionName] = collection
			collectionsLock.Unlock()
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

func honeycombService() *siesta.Service {
	service := siesta.NewService("/1/batch")
	service.Route("POST", "/:collection", "Post an event", func(w http.ResponseWriter, r *http.Request) {
		var params siesta.Params
		collectionName := params.String("collection", "", "collection name")
		err := params.Parse(r.Form)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		var body io.Reader = r.Body

		if r.Header.Get("Content-Encoding") == "gzip" {
			gzipReader, err := gzip.NewReader(r.Body)
			if err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			body = gzipReader
		}

		type payloadElem struct {
			Time string `json:"time"`
			Data Event  `json:"data"`
		}
		var payload []payloadElem
		err = json.NewDecoder(body).Decode(&payload)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		events := []Event{}
		for _, p := range payload {
			e := p.Data
			e["_ts"] = p.Time
			e["_tag"] = "rds"
			events = append(events, e)
		}

		collectionsLock.Lock()
		defer collectionsLock.Unlock()

		collection, present := Collections[*collectionName]
		if !present {
			collection, err = OpenEventCollection(filepath.Join(DataDir, *collectionName+".lm2"))
			if err != nil {
				if err == ErrDoesNotExist {
					collection, err = CreateEventCollection(filepath.Join(DataDir, *collectionName+".lm2"))
				}
				if err != nil {
					log.Println(err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
			Collections[*collectionName] = collection
		}

		err = collection.StoreEvents(events)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Println("stored events", events)
	})
	return service
}
