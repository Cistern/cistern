package snmp

import (
	"bytes"
	"encoding/asn1"
)

type DataType interface {
	Encode() []byte
}

type Sequence []DataType

func (s Sequence) Encode() []byte {
	buf := &bytes.Buffer{}

	for _, entry := range s {
		buf.Write(entry.Encode())
	}

	seqLength := buf.Len()

	return append(encodeHeaderSequence(0x30, seqLength), buf.Bytes()...)
}

type Int int

func (i Int) Encode() []byte {
	b, _ := asn1.Marshal(i)
	return b
}

type String string

func (s String) Encode() []byte {
	return append(encodeHeaderSequence(0x4, len(s)), []byte(s)...)
}

type GetRequest []DataType

func (r GetRequest) Encode() []byte {
	buf := &bytes.Buffer{}

	for _, entry := range r {
		buf.Write(entry.Encode())
	}

	return append(encodeHeaderSequence(0xa0, buf.Len()), buf.Bytes()...)
}

type GetNextRequest []DataType

func (r GetNextRequest) Encode() []byte {
	buf := &bytes.Buffer{}

	for _, entry := range r {
		buf.Write(entry.Encode())
	}

	return append(encodeHeaderSequence(0xa1, buf.Len()), buf.Bytes()...)
}

type GetResponse []DataType

func (r GetResponse) Encode() []byte {
	buf := &bytes.Buffer{}

	for _, entry := range r {
		buf.Write(entry.Encode())
	}

	return append(encodeHeaderSequence(0xa2, buf.Len()), buf.Bytes()...)
}

type Report []DataType

func (r Report) Encode() []byte {
	buf := &bytes.Buffer{}

	for _, entry := range r {
		buf.Write(entry.Encode())
	}

	return append(encodeHeaderSequence(0xa8, buf.Len()), buf.Bytes()...)
}

type ObjectIdentifier []byte

func (oid ObjectIdentifier) Encode() []byte {
	return append(encodeHeaderSequence(0x6, len(oid)), oid...)
}

type null byte

func (n null) Encode() []byte {
	return []byte{0x05, 0}
}

const Null null = 0
