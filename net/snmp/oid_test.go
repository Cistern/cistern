package snmp

import (
	"bytes"
	"testing"
)

func TestEncodeOID(t *testing.T) {
	oid := ObjectIdentifier{1, 3, 6, 1, 4, 1, 2636, 3, 2, 3, 1, 20}

	b, err := oid.Encode()
	if err != nil {
		t.Fatal(err)
	}

	if expected := []byte{
		0x6, 0x0c,

		0x2b, 0x06, 0x01, 0x04,
		0x01, 0x94, 0x4c, 0x03,
		0x02, 0x03, 0x01, 0x14,
	}; bytes.Compare(expected, b) != 0 {
		t.Errorf("encoded ObjectIdentifer incorrect. Expected %v, got %v", expected, b)
	}
}
