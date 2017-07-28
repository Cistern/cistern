package main

import "time"

func formatTs(ts int64) [8]byte {
	b := [8]byte{
		byte(ts >> (8 * 7)),
		byte(ts >> (8 * 6)),
		byte(ts >> (8 * 5)),
		byte(ts >> (8 * 4)),
		byte(ts >> (8 * 3)),
		byte(ts >> (8 * 2)),
		byte(ts >> (8 * 1)),
		byte(ts),
	}
	return b
}

func parseTs(b [8]byte) int64 {
	ts := int64(b[7])
	ts |= int64(b[6]) << (8 * 1)
	ts |= int64(b[5]) << (8 * 2)
	ts |= int64(b[4]) << (8 * 3)
	ts |= int64(b[3]) << (8 * 4)
	ts |= int64(b[2]) << (8 * 5)
	ts |= int64(b[1]) << (8 * 6)
	ts |= int64(b[0]) << (8 * 7)
	return ts
}

func toMicrosecondTime(t time.Time) int64 {
	return t.Unix()*1000000 + int64(t.Nanosecond())/1000
}

func fromMicrosecondTime(t int64) time.Time {
	return time.Unix(t/1000000, (t%1e6)*1000).UTC()
}
