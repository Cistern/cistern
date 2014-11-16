package snmp

import (
	"encoding/asn1"
	"io"
)

func Decode(r io.Reader) (DataType, int) {
	bytesRead := 0

	typeLength := []byte{0, 0}
	n, _ := r.Read(typeLength)

	bytesRead += n

	t := typeLength[0]
	length := int(typeLength[1])

	if length > 0x7F {
		lengthNumBytes := 0x80 ^ byte(length)
		length = 0
		for lengthNumBytes > 0 {
			length = length << 8
			var b [1]byte
			r.Read(b[:])
			length |= int(b[0])

			lengthNumBytes--
		}

	}

	if t == 0x30 {
		seq := Sequence{}
		seqBytes := 0

		for seqBytes < length {
			item, read := Decode(r)
			if read > 0 && item != nil {
				seq = append(seq, item)
				bytesRead += read
				seqBytes += read
			} else {
				break
			}
		}

		return seq, bytesRead
	}

	if t == 0x02 || t == 0x41 {
		intBytes := make([]byte, int(length))
		n, _ := r.Read(intBytes)

		intBytes = append([]byte{0x02, byte(length)}, intBytes...)

		bytesRead += n

		i := 0
		asn1.Unmarshal(intBytes, &i)

		return Int(i), bytesRead
	}

	if t == 0x04 {

		str := make([]byte, length)
		n, _ := r.Read(str)
		bytesRead += n

		return String(str), bytesRead
	}

	if t == 0xa2 {

		res := GetResponse{}
		seqBytes := 0

		for seqBytes < length {
			item, read := Decode(r)
			if read > 0 && item != nil {
				res = append(res, item)
				bytesRead += read
				seqBytes += read
			} else {
				break
			}
		}

		return res, bytesRead
	}

	if t == 0xa8 {

		res := Report{}
		seqBytes := 0

		for seqBytes < length {
			item, read := Decode(r)
			if read > 0 && item != nil {
				res = append(res, item)
				bytesRead += read
				seqBytes += read
			} else {
				break
			}
		}

		return res, bytesRead
	}

	if t == 0x06 {

		oid := make(ObjectIdentifier, length)
		n, _ := r.Read(oid)
		bytesRead += n

		return oid, bytesRead
	}

	return nil, bytesRead
}
