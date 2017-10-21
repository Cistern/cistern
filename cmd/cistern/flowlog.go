package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

// version account-id interface-id srcaddr dstaddr srcport dstport protocol packets bytes start end action log-status

type FlowLogRecord struct {
	Version       string    `json:"version"`
	AccountID     string    `json:"account_id"`
	InterfaceID   string    `json:"interface_id"`
	SourceAddress net.IP    `json:"source_address"`
	DestAddress   net.IP    `json:"dest_address"`
	SourcePort    int       `json:"source_port"`
	DestPort      int       `json:"dest_port"`
	Protocol      int       `json:"protocol"`
	Packets       int       `json:"packets"`
	Bytes         int       `json:"bytes"`
	Start         time.Time `json:"start"`
	End           time.Time `json:"end"`
	Action        string    `json:"action"`
	LogStatus     string    `json:"log_status"`

	Timestamp  time.Time `json:"_ts"`
	Duration   float64   `json:"_duration"`
	StreamName string    `json:"stream_name"`
}

func (r *FlowLogRecord) Parse(s string) error {
	parts := strings.Split(s, " ")
	if len(parts) != 14 {
		fmt.Println(parts)
		return errors.New("invalid flow log record")
	}

	r.Version = parts[0]
	r.AccountID = parts[1]
	r.InterfaceID = parts[2]
	r.SourceAddress = net.ParseIP(parts[3])
	r.DestAddress = net.ParseIP(parts[4])

	n, err := strconv.ParseInt(parts[5], 10, 64)
	if err != nil {
		return err
	}

	r.SourcePort = int(n)

	n, err = strconv.ParseInt(parts[6], 10, 64)
	if err != nil {
		return err
	}
	r.DestPort = int(n)

	n, err = strconv.ParseInt(parts[7], 10, 64)
	if err != nil {
		return err
	}
	r.Protocol = int(n)

	n, err = strconv.ParseInt(parts[8], 10, 64)
	if err != nil {
		return err
	}
	r.Packets = int(n)

	n, err = strconv.ParseInt(parts[9], 10, 64)
	if err != nil {
		return err
	}
	r.Bytes = int(n)

	n, err = strconv.ParseInt(parts[10], 10, 64)
	if err != nil {
		return err
	}
	r.Start = time.Unix(n, 0).UTC()

	n, err = strconv.ParseInt(parts[11], 10, 64)
	if err != nil {
		return err
	}
	r.End = time.Unix(n, 0).UTC()

	r.Action = parts[12]
	r.LogStatus = parts[13]

	return nil
}

func (r *FlowLogRecord) ToEvent() Event {
	return Event{
		"version":        r.Version,
		"account_id":     r.AccountID,
		"interface_id":   r.InterfaceID,
		"source_address": r.SourceAddress,
		"dest_address":   r.DestAddress,
		"source_port":    r.SourcePort,
		"dest_port":      r.DestPort,
		"protocol":       r.Protocol,
		"packets":        r.Packets,
		"bytes":          r.Bytes,
		"start":          r.Start.Format(time.RFC3339Nano),
		"end":            r.End.Format(time.RFC3339Nano),
		"action":         r.Action,
		"log_status":     r.LogStatus,
		"_ts":            r.Timestamp.Format(time.RFC3339Nano),
		"_tag":           r.StreamName,
	}
}

func captureFlowLogs(groupName string, retention int, done chan struct{}) error {
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
	currentBatch := []*FlowLogRecord{}
	timer := time.NewTimer(0)

	log.Println("Starting poll of flow log group", groupName)

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
			log.Println("Stopping poll of flow log group", groupName)
			eventCollection.col.Close()
			return nil
		}

		logEvents, err := cwl.GetLogEvents(nextBatchStart)
		if err != nil {
			return err
		}

		for _, e := range logEvents {
			rec := &FlowLogRecord{}
			err = rec.Parse(*e.Message)
			if err == nil {
				rec.Timestamp = rec.Start
				rec.Duration = rec.End.Sub(rec.Start).Seconds()
				rec.StreamName = *e.LogStreamName
				currentBatch = append(currentBatch, rec)
			}

			if nextBatchStart < *e.Timestamp {
				nextBatchStart = *e.Timestamp
			}
		}

		events := []Event{}
		for _, rec := range currentBatch {
			event := rec.ToEvent()
			event["_tag"] = rec.StreamName
			events = append(events, event)
		}

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
