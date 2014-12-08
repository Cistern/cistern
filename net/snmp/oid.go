package snmp

import (
	"strconv"
	"strings"
)

func ParseOID(str string) (ObjectIdentifier, error) {
	parts := strings.Split(strings.Trim(str, "."), ".")

	oid := ObjectIdentifier{}

	for _, part := range parts {
		n, err := strconv.ParseUint(part, 10, 16)
		if err != nil {
			return nil, err
		}

		oid = append(oid, uint16(n))
	}

	return oid, nil
}

func MustParseOID(str string) ObjectIdentifier {
	oid, err := ParseOID(str)
	if err != nil {
		panic(err)
	}

	return oid
}

func encodeOIDUint(i uint16) []byte {
	var b []byte

	if i < 128 {
		return []byte{byte(i)}
	}

	b = append(b, byte(i)%128)
	i /= 128

	for i > 0 {
		b = append(b, 128+byte(i)%128)
		i /= 128
	}

	return reverseSlice(b)
}

func reverseSlice(b []byte) []byte {
	length := len(b)
	result := make([]byte, 0, length)

	for length > 0 {
		result = append(result, b[length-1])
		length--
	}

	return result
}
