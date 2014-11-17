package snmp

import (
	"bytes"
	"encoding/asn1"
)

type DataType interface {
	Encode() ([]byte, error)
}

type Sequence []DataType

func (s Sequence) Encode() ([]byte, error) {
	buf := &bytes.Buffer{}

	for _, entry := range s {
		encodedEntry, err := entry.Encode()
		if err != nil {
			return nil, err
		}

		_, err = buf.Write(encodedEntry)
		if err != nil {
			return nil, err
		}
	}

	seqLength := buf.Len()

	return append(encodeHeaderSequence(0x30, seqLength), buf.Bytes()...), nil
}

type Int int

func (i Int) Encode() ([]byte, error) {
	return asn1.Marshal(i)
}

type String string

func (s String) Encode() ([]byte, error) {
	return append(encodeHeaderSequence(0x4, len(s)), []byte(s)...), nil
}

type GetRequest []DataType

func (s GetRequest) Encode() ([]byte, error) {
	buf := &bytes.Buffer{}

	for _, entry := range s {
		encodedEntry, err := entry.Encode()
		if err != nil {
			return nil, err
		}

		_, err = buf.Write(encodedEntry)
		if err != nil {
			return nil, err
		}
	}

	seqLength := buf.Len()

	return append(encodeHeaderSequence(0xa0, seqLength), buf.Bytes()...), nil
}

type GetNextRequest []DataType

func (s GetNextRequest) Encode() ([]byte, error) {
	buf := &bytes.Buffer{}

	for _, entry := range s {
		encodedEntry, err := entry.Encode()
		if err != nil {
			return nil, err
		}

		_, err = buf.Write(encodedEntry)
		if err != nil {
			return nil, err
		}
	}

	seqLength := buf.Len()

	return append(encodeHeaderSequence(0xa1, seqLength), buf.Bytes()...), nil
}

type GetResponse []DataType

func (s GetResponse) Encode() ([]byte, error) {
	buf := &bytes.Buffer{}

	for _, entry := range s {
		encodedEntry, err := entry.Encode()
		if err != nil {
			return nil, err
		}

		_, err = buf.Write(encodedEntry)
		if err != nil {
			return nil, err
		}
	}

	seqLength := buf.Len()

	return append(encodeHeaderSequence(0xa2, seqLength), buf.Bytes()...), nil
}

type Report []DataType

func (s Report) Encode() ([]byte, error) {
	buf := &bytes.Buffer{}

	for _, entry := range s {
		encodedEntry, err := entry.Encode()
		if err != nil {
			return nil, err
		}

		_, err = buf.Write(encodedEntry)
		if err != nil {
			return nil, err
		}
	}

	seqLength := buf.Len()

	return append(encodeHeaderSequence(0xa8, seqLength), buf.Bytes()...), nil
}

type ObjectIdentifier []byte

func (oid ObjectIdentifier) Encode() ([]byte, error) {
	return append(encodeHeaderSequence(0x6, len(oid)), oid...), nil
}

type null byte

func (n null) Encode() ([]byte, error) {
	return []byte{0x05, 0}, nil
}

const Null null = 0
