package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Preetam/siesta"
)

func service() *siesta.Service {
	service := siesta.NewService("/")
	service.Route("POST", "/collections/:collection/query", "query a collection", func(w http.ResponseWriter, r *http.Request) {
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

		queryDesc := QueryDesc{}
		err = json.NewDecoder(r.Body).Decode(&queryDesc)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		result, err := collection.Query(queryDesc)
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
