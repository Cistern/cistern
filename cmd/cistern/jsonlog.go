package main

import (
	"encoding/json"
	"log"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

func captureJSONLogs(groupName string, retention int, done chan struct{}) error {
	if retention == 0 {
		retention = 7
		log.Printf("Missing retention for %s; defaulting to %d days", groupName, retention)
	}
	stop := make(chan struct{}, 1)

	go func() {
		<-done
		stop <- struct{}{}
	}()

	cwl, err := NewCloudWatchLog(cloudwatchlogs.New(session.Must(session.NewSession())), groupName)
	if err != nil {
		return err
	}

	collectionsLock.Lock()
	eventCollection := Collections[groupName]
	if eventCollection == nil {
		eventCollection, err = OpenEventCollection(filepath.Join(DataDir, groupName+".lm2"))
		if err != nil {
			if err == ErrDoesNotExist {
				eventCollection, err = CreateEventCollection(filepath.Join(DataDir, groupName+".lm2"))
			}
			if err != nil {
				collectionsLock.Unlock()
				return err
			}
		}
		Collections[groupName] = eventCollection
	}
	collectionsLock.Unlock()

	eventCollection.SetRetention(retention)

	nextBatchStart := cwl.LastTimestamp()
	currentBatch := []Event{}
	timer := time.NewTimer(0)

	log.Println("Starting poll of JSON log group", groupName)

	for {
		lastTime := time.Unix((nextBatchStart)/1000, 0)
		if time.Now().Sub(lastTime) <= 5*time.Minute {
			// Last event was within 5 minutes, so wait a minute
			// before next poll.
			timer.Reset(time.Minute)
		} else {
			// Catching up, so only wait 5 seconds.
			timer.Reset(5 * time.Second)
		}

		select {
		case <-timer.C:
		case <-stop:
			log.Println("Stopping poll of JSON log group", groupName)
			eventCollection.col.Close()
			return nil
		}

		logEvents, err := cwl.GetLogEvents(nextBatchStart)
		if err != nil {
			return err
		}

		for _, e := range logEvents {
			event := Event{}
			err = json.Unmarshal([]byte(*e.Message), &event)
			if err == nil {
				timestamp := time.Unix(*e.Timestamp/1000, (*e.Timestamp%1000)*1000000)
				event["_ts"] = timestamp.Format(time.RFC3339Nano)
				event["_tag"] = *e.LogStreamName
				currentBatch = append(currentBatch, event)

				if nextBatchStart < *e.Timestamp {
					nextBatchStart = *e.Timestamp
				}
			}
		}

		events := currentBatch
		if len(logEvents) > 0 {
			log.Printf("Logs group %s: aggregated %d events", groupName, len(logEvents))
			err = eventCollection.StoreEvents(events)
			if err != nil {
				return err
			}
			nextBatchStart += 1
			err = cwl.SetLastTimestamp(nextBatchStart)
			if err != nil {
				return err
			}
		}

		currentBatch = currentBatch[:0]
	}
}
