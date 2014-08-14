package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/PreetamJinka/listmap"
)

type MetricStorage struct {
	lm *listmap.Listmap
}

func OpenOrCreateStorage(loc string) (*MetricStorage, error) {
	lm := listmap.OpenListmap(loc)
	if lm == nil {
		lm = listmap.NewListmap(loc)
		if lm == nil {
			return nil, errors.New("could not open or create listmap")
		}
	}

	return &MetricStorage{
		lm: lm,
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
	key.WriteString("obs-" + host + "\x00" + metric + "\x00")
	binary.Write(key, binary.BigEndian, uint64(ts.Unix()))
	binary.Write(val, binary.LittleEndian, value)
	m.lm.Set(key.Bytes(), val.Bytes())
}

func (m *MetricStorage) String() string {
	str := ""
	for cur := m.lm.NewCursor(); cur != nil; cur = cur.Next() {
		if bytes.Compare(cur.Key(), []byte("obs-")) > 0 {
			if bytes.Compare(cur.Key(), []byte("obs-\xff")) >= 0 {
				break
			}

			host, metric, ts, value := decodeKeyValue(cur.Key(), cur.Value())
			str += fmt.Sprintf("  %s:%s[%d] = %f\n", host, metric, ts.Unix(), value)
		}
	}

	return str
}

func decodeKeyValue(key, value []byte) (string, string, time.Time, float32) {
	host := ""
	metric := ""
	var ts time.Time

	i := 0
	for i = 0; i < len(key); i++ {
		if key[i] == 0 {
			host = string(key[:i])
			break
		}
	}

	key = key[i+1:]

	for i = 0; i < len(key); i++ {
		if key[i] == 0 {
			metric = string(key[:i])
			break
		}
	}

	key = key[i+1:]

	var unixTime uint64
	binary.Read(bytes.NewReader(key), binary.BigEndian, &unixTime)
	ts = time.Unix(int64(unixTime), 0)

	var val float32
	binary.Read(bytes.NewReader(value), binary.LittleEndian, &val)

	return host, metric, ts, val
}
