package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

var version = "0.1.0"

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	address := flag.String("address", "http://localhost:2020", "Cistern node address")
	collection := flag.String("collection", "", "Collection to query")
	columns := flag.String("columns", "", "Comma-separated list of columns to aggregate. Example: 'sum(bytes), sum(packets)'")
	group := flag.String("group", "", "Comma-separated list of fields to group by. Example: 'source_address, dest_address'")
	filters := flag.String("filters", "", "Comma-separated list of filters. Filters have the format '<column> <condition> <value>'."+
		" Possible conditions are [eq,neq]. Values have to be valid JSON values. Example: 'dest_address neq \"172.31.31.192\" , packets eq 3'")
	start := flag.Int64("start", time.Now().Unix()-3600, "Start Unix timestamp")
	end := flag.Int64("end", time.Now().Unix(), "End Unix timestamp")
	orderBy := flag.String("order-by", "", "Comma-separated list of columns to order by."+
		" Providing multiple columns means the results are ordered by the first column, then the next, etc.")
	limit := flag.Int("limit", 0, "Maximum number of events to return.")
	pointSize := flag.Duration("point-size", 0, "Point size of time series. 0 means series will not be generated.")
	descending := flag.Bool("descending", false, "Sort in descending order.")
	showVersion := flag.Bool("version", false, "Show version and exit.")
	flag.Parse()

	if *showVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	queryDesc, err := parseQuery(*columns, *group, *filters, *orderBy)
	if err != nil {
		log.Fatalln(err)
	}

	queryDesc.TimeRange.Start = time.Unix(*start, 0)
	queryDesc.TimeRange.End = time.Unix(*end, 0)
	queryDesc.Limit = *limit
	queryDesc.PointSize = (*pointSize).Nanoseconds() / 1000
	queryDesc.Descending = *descending
	buf := &bytes.Buffer{}
	err = json.NewEncoder(buf).Encode(queryDesc)
	if err != nil {
		log.Fatalln(err)
	}
	response, err := http.Post(fmt.Sprintf("%s/collections/%s/query", *address, *collection), "application/json", buf)
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
