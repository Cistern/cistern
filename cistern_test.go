package main

import (
	"testing"
)

func TestFloatConv(t *testing.T) {
	assert := func(f, expected float32) {
		if f != expected {
			t.Error(f, " != ", expected)
		}
	}

	assert(getFloat32Value(uint64(1234560)), 1234560)
	assert(getFloat32Value(uint32(1234561)), 1234561)
	assert(getFloat32Value(int64(-1234562)), -1234562)
	assert(getFloat32Value(uint(1234563)), 1234563)
	assert(getFloat32Value(int32(1234564)), 1234564)
}
