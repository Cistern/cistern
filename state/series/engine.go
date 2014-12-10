package series

import (
	"bytes"
	"encoding/binary"
	"errors"
	"time"

	"github.com/PreetamJinka/cistern/state/series/bolt"
)

type Engine struct {
	db *bolt.DB
}

func NewEngine(filename string) (*Engine, error) {
	db, err := bolt.Open(filename, 0666, nil)
	if err != nil {
		return nil, err
	}

	return &Engine{
		db: db,
	}, nil
}

func (e *Engine) InsertPoint(host, metric string, ts time.Time, value float32) error {
	return e.db.Update(func(tx *bolt.Tx) error {
		hostBucket, err := tx.CreateBucketIfNotExists([]byte(host))
		if err != nil {
			return err
		}

		metricBucket, err := hostBucket.CreateBucketIfNotExists([]byte(metric))
		if err != nil {
			return err
		}

		key := &bytes.Buffer{}
		val := &bytes.Buffer{}

		err = binary.Write(key, binary.BigEndian, uint64(ts.Unix()))
		if err != nil {
			return err
		}

		err = binary.Write(val, binary.LittleEndian, value)
		if err != nil {
			return err
		}

		return metricBucket.Put(key.Bytes(), val.Bytes())
	})
}

type point struct {
	Timestamp int64   `json:"ts"`
	Value     float32 `json:"value"`
}

func (e *Engine) Query(host, metric string) []point {
	result := []point{}

	e.db.View(func(tx *bolt.Tx) error {
		hostBucket := tx.Bucket([]byte(host))

		if hostBucket == nil {
			return errors.New("unknown host")
		}

		metricBucket := hostBucket.Bucket([]byte(metric))
		if metricBucket == nil {
			return errors.New("unknown metric")
		}

		c := metricBucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			ts, val := decodeKeyValue(k, v)

			result = append(result, point{
				Timestamp: ts.Unix(),
				Value:     val,
			})
		}

		return nil
	})

	return result
}

func decodeKeyValue(key, value []byte) (time.Time, float32) {
	var ts time.Time

	var unixTime uint64
	binary.Read(bytes.NewReader(key), binary.BigEndian, &unixTime)
	ts = time.Unix(int64(unixTime), 0)

	var val float32
	binary.Read(bytes.NewReader(value), binary.LittleEndian, &val)

	return ts, val
}
