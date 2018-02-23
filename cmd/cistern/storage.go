package main

import (
	"encoding/json"
	"errors"
	"regexp"
	"sync"
	"time"

	"github.com/Preetam/lm2"
)

var (
	ErrDoesNotExist = errors.New("cistern: does not exist")

	eventIDTagRegexp = regexp.MustCompile("^[a-zA-Z0-9_./-]{1,256}$")

	minTimestamp = time.Unix(0, 0)
)

const (
	eventKeyPrefix byte = 'e'
)

// Event represents a flat JSON event object.
type Event map[string]interface{}

type EventCollection struct {
	filename  string
	col       *lm2.Collection
	retention int // event retention in days
	lock      sync.RWMutex
}

func OpenEventCollection(filename string) (*EventCollection, error) {
	col, err := lm2.OpenCollection(filename, 10000000000)
	if err != nil {
		if err == lm2.ErrDoesNotExist {
			return nil, ErrDoesNotExist
		}
		return nil, err
	}
	return &EventCollection{
		filename: filename,
		col:      col,
	}, nil
}

func CreateEventCollection(filename string) (*EventCollection, error) {
	col, err := lm2.NewCollection(filename, 10000000000)
	if err != nil {
		return nil, err
	}
	return &EventCollection{
		filename: filename,
		col:      col,
	}, nil
}

func (c *EventCollection) SetRetention(days int) {
	c.retention = days
}

func (c *EventCollection) StoreEvents(events []Event) error {
	c.lock.RLock()
	defer c.lock.RUnlock()

	// Validate tags
	for _, e := range events {
		tag, ok := e["_tag"].(string)
		if !ok {
			return errors.New("invalid tag")
		}
		if !eventIDTagRegexp.MatchString(tag) {
			return errors.New("invalid tag")
		}
	}

	wb := lm2.NewWriteBatch()
	for _, event := range events {
		delete(event, "_id")
		tag := event["_tag"].(string)

		marshalled, err := json.Marshal(event)
		if err != nil {
			return err
		}

		var ts int64
		if tsVal, ok := event["_ts"]; ok {
			if tsString, ok := tsVal.(string); ok {
				timeTs, err := time.Parse(time.RFC3339Nano, tsString)
				if err != nil {
					return errors.New("ts is not formatted per RFC 3339")
				}
				if timeTs.Before(minTimestamp) {
					return errors.New("ts before Unix epoch")
				}
				ts = toMicrosecondTime(timeTs)
			} else {
				return errors.New("ts is not a string")
			}
		} else {
			return errors.New("missing event ts")
		}

		hash := ""
		if hashValue, ok := event["_hash"]; ok {
			if hashString, ok := hashValue.(string); ok {
				hash = hashString
			}
		}

		formattedTs := formatTs(ts)
		idStr := string(eventKeyPrefix) + string(formattedTs[:]) + "|" + tag + "|" + hash
		wb.Set(idStr, string(marshalled))
	}

	_, err := c.col.Update(wb)
	if err != nil {
		return err
	}

	return nil
}

func (c *EventCollection) Compact() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	minTs := time.Now().Add(-1 * time.Duration(c.retention) * 24 * time.Hour)
	err := c.col.CompactFunc(func(key string, value string) (string, string, bool) {
		if key[0] != eventKeyPrefix {
			return key, value, true
		}

		ts, _, _, err := splitCollectionID(key)
		if err != nil {
			return key, value, true
		}
		if fromMicrosecondTime(ts).Before(minTs) {
			return "", "", false
		}

		return key, value, true
	})
	if err != nil {
		return err
	}

	col, err := lm2.OpenCollection(c.filename, 10000000000)
	if err != nil {
		if err == lm2.ErrDoesNotExist {
			return ErrDoesNotExist
		}
		return err
	}
	c.col = col
	return nil
}
