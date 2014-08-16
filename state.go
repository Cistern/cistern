package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/boltdb/bolt"
)

type MetricStorage struct {
	bolt *bolt.DB
}

func OpenOrCreateStorage(loc string) (*MetricStorage, error) {
	db, err := bolt.Open(loc, 0600, nil)
	if err != nil {
		return nil, err
	}

	return &MetricStorage{
		bolt: db,
	}, nil
}

func (m *MetricStorage) SnapshotRegistry(r *HostRegistry) {
	r.lock.RLock()
	defer r.lock.RUnlock()

	for host, metrics := range r.hosts {
		for metric, metricState := range metrics.metrics {
			m.StoreMetricObservation(host, metric, time.Now(), metricState.Value())
		}
	}
}

func (m *MetricStorage) StoreMetricObservation(host, metric string, ts time.Time, value float32) {
	key := &bytes.Buffer{}
	val := &bytes.Buffer{}
	key.WriteString(metric + "\x00")
	binary.Write(key, binary.BigEndian, uint64(ts.Unix()))
	binary.Write(val, binary.LittleEndian, value)

	m.bolt.Update(func(tx *bolt.Tx) error {
		hostsBucket, err := tx.CreateBucketIfNotExists([]byte("hosts"))
		if err != nil {
			return err
		}

		err = hostsBucket.Put([]byte(host), nil)
		if err != nil {
			return err
		}

		b, err := tx.CreateBucketIfNotExists([]byte(host))
		if err != nil {
			return err
		}

		err = b.Put(key.Bytes(), val.Bytes())

		return err
	})
}

func (m *MetricStorage) String() string {
	str := ""
	var hosts [][]byte
	m.bolt.View(func(tx *bolt.Tx) error {
		hostsBucket := tx.Bucket([]byte("hosts"))
		if hostsBucket == nil {
			return nil
		}
		c := hostsBucket.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			hosts = append(hosts, k)
		}
		return nil
	})

	for _, host := range hosts {
		m.bolt.View(func(tx *bolt.Tx) error {
			c := tx.Bucket(host).Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				metric, ts, value := decodeKeyValue(k, v)
				str += fmt.Sprintf("  %s:%s[%d] = %f\n", string(host), metric, ts.Unix(), value)
			}
			return nil

		})
	}

	return str
}

func decodeKeyValue(key, value []byte) (string, time.Time, float32) {
	metric := ""
	var ts time.Time

	for i := 0; i < len(key)-1; i++ {
		if key[i] == 0 {
			metric = string(key[:i])
			key = key[i+1:]
			break
		}
	}

	var unixTime uint64
	binary.Read(bytes.NewReader(key), binary.BigEndian, &unixTime)
	ts = time.Unix(int64(unixTime), 0)

	var val float32
	binary.Read(bytes.NewReader(value), binary.LittleEndian, &val)

	return metric, ts, val
}
