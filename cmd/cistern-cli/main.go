package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

var version = "0.1.1"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	address := flag.String("address", "http://localhost:2020", "Cistern node address")
	collection := flag.String("collection", "", "Collection to query")
	start := flag.Int64("start", time.Now().Unix()-3600, "Start Unix timestamp")
	end := flag.Int64("end", time.Now().Unix(), "End Unix timestamp")
	query := flag.String("query", "", "Query string")
	showVersion := flag.Bool("version", false, "Show version and exit.")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	endpoint := fmt.Sprintf("%s/collections/%s/query?start=%d&end=%d&query=%s",
		*address, *collection, *start, *end, url.QueryEscape(*query))
	response, err := http.Post(endpoint, "application/json", nil)
	if err != nil {
		log.Fatalln(err)
	}

	queryResult := QueryResult{}
	err = json.NewDecoder(response.Body).Decode(&queryResult)
	if err != nil {
		log.Fatalln(err)
	}

	pretty, err := json.MarshalIndent(queryResult, "", "  ")
	if err != nil {
		log.Fatalln(err)
	}

	fmt.Printf("%s\n", pretty)
}
