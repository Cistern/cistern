package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

type FlowLogState struct {
	LastTimestamp int64  `json:"last_timestamp"`
	LastEventID   string `json:"last_event"`

	filename string
}

func (s *FlowLogState) Store() error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(s.filename, data, 0600)
	return err
}

func (s *FlowLogState) Load() error {
	data, err := ioutil.ReadFile(s.filename)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, s)
	return err
}

func NewFlowLogState(filename string) (*FlowLogState, error) {
	state := &FlowLogState{
		filename: filename,
	}
	state.Load()
	return state, state.Store()
}

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
		"source_address": r.SourceAddress,
		"dest_address":   r.DestAddress,
		"source_port":    r.SourcePort,
		"dest_port":      r.DestPort,
		"protocol":       r.Protocol,
		"packets":        r.Packets,
		"bytes":          r.Bytes,
		"_ts":            r.Timestamp.Format(time.RFC3339Nano),
		"_tag":           r.StreamName,
	}
}

func groupFlowRecords(records []*FlowLogRecord) []FlowLogRecord {
	type groupKey struct {
		Timestamp     time.Time
		SourceAddress [16]byte
		DestAddress   [16]byte
		SourcePort    int
		DestPort      int
		Protocol      int
	}
	groups := map[groupKey]*FlowLogRecord{}
	for _, rec := range records {
		key := groupKey{
			Timestamp:     rec.Timestamp.Truncate(time.Minute * 10),
			SourceAddress: ipTo16Bytes(rec.SourceAddress),
			DestAddress:   ipTo16Bytes(rec.DestAddress),
			SourcePort:    rec.SourcePort,
			DestPort:      rec.DestPort,
			Protocol:      rec.Protocol,
		}
		groupRec := groups[key]
		if groupRec == nil {
			groupRec = &FlowLogRecord{
				Timestamp:     key.Timestamp,
				SourceAddress: net.IP(key.SourceAddress[:]),
				DestAddress:   net.IP(key.DestAddress[:]),
				SourcePort:    key.SourcePort,
				DestPort:      key.DestPort,
				Protocol:      key.Protocol,
			}
			groups[key] = groupRec
		}
		groupRec.Bytes += rec.Bytes
		groupRec.Packets += rec.Packets
	}

	result := []FlowLogRecord{}
	for _, rec := range groups {
		result = append(result, *rec)
	}
	return result
}

func ipTo16Bytes(ip net.IP) [16]byte {
	result := [16]byte{}
	copy(result[:], ip.To16())
	return result
}

func captureFlowLogs(groupName string, retention int, done chan struct{}) error {
	if retention == 0 {
		retention = 3
		log.Printf("Missing retention for %s; defaulting to %d days", groupName, retention)
	}
	stop := make(chan struct{}, 1)

	go func() {
		<-done
		stop <- struct{}{}
	}()

	stateFile, err := NewFlowLogState(filepath.Join(DataDir, groupName+".state"))
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

	nextBatchStart := stateFile.LastTimestamp
	if nextBatchStart == 0 {
		nextBatchStart = (time.Now().Unix() - 86400) * 1000
		nextBatchStart = (nextBatchStart / regularBatchSize) * regularBatchSize
	}

	currentBatch := []*FlowLogRecord{}
	timer := time.NewTimer(0)

	cwl := NewCloudWatchLog(cloudwatchlogs.New(session.Must(session.NewSession())), groupName)

	for {
		select {
		case <-timer.C:
		case <-stop:
			log.Println("stopping", groupName)
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
			rec := &FlowLogRecord{}
			err = rec.Parse(*e.Message)
			if err == nil {
				rec.Timestamp = rec.Start
				rec.Duration = rec.End.Sub(rec.Start).Seconds()
				rec.StreamName = *e.LogStreamName
				currentBatch = append(currentBatch, rec)
			}
		}

		grouped := groupFlowRecords(currentBatch)
		events := []Event{}
		for _, rec := range grouped {
			event := rec.ToEvent()
			event["_tag"] = "flowlog"
			events = append(events, event)
		}

		if len(logEvents) > 0 {
			log.Printf("Flow Logs group %s: aggregated %d events", groupName, len(logEvents))
			err = eventCollection.StoreEvents(events)
			if err != nil {
				return err
			}

			stateFile.LastTimestamp = nextBatchStart
			stateFile.Store()
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
