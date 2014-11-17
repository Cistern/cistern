package snmp

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/binary"
)

func passphraseToKey(passphrase, engineId []byte) []byte {
	h := sha1.New()

	passphraseLength := len(passphrase)
	repeat, remain := 1048576/passphraseLength, 1048576%passphraseLength

	for repeat > 0 {
		h.Write(passphrase)
		repeat--
	}

	if remain > 0 {
		h.Write(passphrase[:remain])
	}

	sum := h.Sum(nil)

	h.Reset()

	h.Write(sum)
	h.Write(engineId)
	h.Write(sum)

	return h.Sum(nil)
}

func (s *Session) encrypt(payload []byte) ([]byte, []byte) {
	b := &bytes.Buffer{}
	binary.Write(b, binary.BigEndian, s.engineBoots)
	binary.Write(b, binary.BigEndian, s.engineTime)

	b2 := &bytes.Buffer{}
	binary.Write(b2, binary.BigEndian, s.aesIV)
	s.aesIV++

	priv := b2.Bytes()
	iv := append(b.Bytes(), priv...)

	encrypted := make([]byte, len(payload))

	aesBlockEncrypter, err := aes.NewCipher(s.privKey)
	if err != nil {
		return nil, nil
	}

	aesEncrypter := cipher.NewCFBEncrypter(aesBlockEncrypter, iv)
	aesEncrypter.XORKeyStream(encrypted, payload)

	return encrypted, priv
}

func (s *Session) decrypt(payload, priv []byte) []byte {
	b := &bytes.Buffer{}
	binary.Write(b, binary.BigEndian, s.engineBoots)
	binary.Write(b, binary.BigEndian, s.engineTime)

	iv := append(b.Bytes(), priv...)

	decrypted := make([]byte, len(payload))

	aesBlockDecrypter, err := aes.NewCipher(s.privKey)
	if err != nil {
		return nil
	}

	aesDecrypter := cipher.NewCFBDecrypter(aesBlockDecrypter, iv)
	aesDecrypter.XORKeyStream(decrypted, payload)

	return decrypted
}

func (s *Session) auth(payload []byte) []byte {
	paddedAuthKey := make([]byte, 64)
	copy(paddedAuthKey, s.authKey)

	a := make([]byte, 64)
	b := make([]byte, 64)

	for i := range a {
		a[i], b[i] = paddedAuthKey[i]^0x36, paddedAuthKey[i]^0x5c
	}

	h := sha1.New()
	h.Write(append(a, payload...))
	tmp := h.Sum(nil)
	h.Reset()

	h.Write(append(b, tmp...))
	return h.Sum(nil)[:12]
}
