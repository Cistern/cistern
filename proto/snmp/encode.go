package snmp

func encodeHeaderSequence(fieldType byte, length int) []byte {
	result := []byte{fieldType}

	if length <= 0x7f {
		result = append(result, byte(length))
	} else {
		result = append(result, 0x80)

		reversed := []byte{}
		for length > 0 {
			reversed = append(reversed, byte(length))
			result = append(result, 0)
			length = length >> 8
		}

		numBytes := len(reversed)
		for i, j := numBytes-1, 2; i >= 0; i, j = i-1, j+1 {
			result[j] = reversed[i]
		}

		result[1] |= byte(numBytes)
	}

	return result
}
