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
	version         = "0.1.1"
)

func main() {
	configFilePath := flag.String("config", "./cistern.json", "Path to config file")
	apiAddr := flag.String("api-addr", "localhost:2020", "API listen address")
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
			go func() {
				err := captureFlowLogs(group.Name, config.Retention, done)
				if err != nil {
					log.Fatal(err)
				}
			}()
		}
	}

	go http.ListenAndServe(*apiAddr, service())

	<-done
	log.Println("Waiting for things to get cleaned up...")
	time.Sleep(time.Second)
}
