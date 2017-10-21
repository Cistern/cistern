package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var (
	DataDir         = "./data/"
	Collections     = map[string]*EventCollection{}
	collectionsLock sync.Mutex
	version         = "0.2.0"
)

func main() {
	configFilePath := flag.String("config", "./cistern.json", "Path to config file")
	apiAddr := flag.String("api-addr", "localhost:2020", "API listen address")
	uiContentPath := flag.String("ui-content", "", "Path to static UI content (enables UI)")
	flag.StringVar(&DataDir, "data-dir", DataDir, "Data directory")
	flag.Parse()

	log.Printf("Cistern v%s starting", version)

	configFileData, err := ioutil.ReadFile(*configFilePath)
	if err != nil {
		log.Fatal("Failed to read config file:", err)
	}

	config := Config{}
	err = json.Unmarshal(configFileData, &config)
	if err != nil {
		log.Fatal("Not a valid config file:", err)
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		sig := <-sigs
		log.Println("Got signal", sig)
		close(done)
	}()

	for _, group := range config.CloudWatchLogs {
		if group.FlowLog {
			go func(group ConfigCloudWatchLogGroup) {
				err := captureFlowLogs(group.Name, config.Retention, done)
				if err != nil {
					log.Fatal(err)
				}
			}(group)
		} else {
			go func(group ConfigCloudWatchLogGroup) {
				err := captureJSONLogs(group.Name, config.Retention, done)
				if err != nil {
					log.Fatal(err)
				}
			}(group)
		}
	}

	if *uiContentPath != "" {
		handler, err := UI(*uiContentPath)
		if err != nil {
			log.Fatalln("Couldn't set up UI:", err)
		}
		http.Handle("/ui/", handler)
	}

	http.Handle("/api/", service())
	go func() {
		log.Println("Listening on", *apiAddr)
		log.Printf("API endpoint is http://%s/api/", *apiAddr)
		if *uiContentPath != "" {
			log.Printf("UI endpoint is http://%s/ui/", *apiAddr)
		}
		err := http.ListenAndServe(*apiAddr, nil)
		if err != nil {
			log.Fatalln("Couldn't start API server:", err)
		}
	}()

	<-done
	log.Println("Waiting for things to get cleaned up...")
	time.Sleep(250 * time.Millisecond)
	log.Println("Exiting.")
}
