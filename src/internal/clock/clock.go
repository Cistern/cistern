package clock

import (
	"log"
	"sync/atomic"
	"time"
)

var now int64

func Run() {
	atomic.StoreInt64(&now, time.Now().Unix())
	log.Println("  Starting internal clock")
	go func() {
		for t := range time.Tick(time.Second) {
			atomic.StoreInt64(&now, t.Unix())
		}
	}()
}

func Time() int64 {
	return atomic.LoadInt64(&now)
}
