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
		retention = 3
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

	const regularBatchSize = 5 * 60 * 1000         // 5 minutes
	const catchupBatchSize = 72 * regularBatchSize // 6 hours

	nextBatchStart := cwl.LastTimestamp()
	if nextBatchStart == 0 {
		nextBatchStart = (time.Now().Unix() - 86400) * 1000
		nextBatchStart = (nextBatchStart / regularBatchSize) * regularBatchSize
	}

	currentBatch := []Event{}
	timer := time.NewTimer(0)

	for {
		select {
		case <-timer.C:
		case <-stop:
			log.Println("Stopping", groupName)
			eventCollection.col.Close()
			return nil
		}

		batchSize := int64(regularBatchSize)
		if now := time.Now().Unix() * 1000; now-nextBatchStart > 6*regularBatchSize {
			batchSize = now - nextBatchStart
			batchSize = (batchSize / regularBatchSize) * regularBatchSize
			if batchSize > catchupBatchSize {
				batchSize = catchupBatchSize
			}
		}

		batchEnd := nextBatchStart + batchSize - 1

		logEvents, err := cwl.GetLogEvents(nextBatchStart, batchEnd)
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
			}
		}

		events := currentBatch
		if len(logEvents) > 0 {
			log.Printf("Logs group %s: aggregated %d events", groupName, len(logEvents))
			err = eventCollection.StoreEvents(events)
			if err != nil {
				return err
			}

			err := cwl.SetLastTimestamp(nextBatchStart)
			if err != nil {
				return err
			}
		}
		next := time.Unix((batchEnd+regularBatchSize)/1000, 0)
		if time.Now().After(next) {
			timer.Reset(0)
		} else {
			timer.Reset(next.Sub(time.Now()) + time.Second)
		}
		nextBatchStart = batchEnd + 1
		currentBatch = currentBatch[:0]
	}
}
