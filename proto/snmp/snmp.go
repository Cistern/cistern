package snmp

import (
	"bytes"
	"errors"
	"log"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"
)

type Session struct {
	addr           *net.UDPAddr
	conn           *net.UDPConn
	user           []byte
	authPassphrase []byte
	privPassphrase []byte

	engineID    []byte
	engineBoots int32
	engineTime  int32

	authKey []byte
	privKey []byte
	aesIV   int64

	inflight map[int]chan DataType
	lock     sync.Mutex
}

func NewSession(address, user, authPassphrase, privPassphrase string) (*Session, error) {
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, err
	}

	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, err
	}

	sess := &Session{
		addr:           addr,
		conn:           conn,
		user:           []byte(user),
		authPassphrase: []byte(authPassphrase),
		privPassphrase: []byte(privPassphrase),
		aesIV:          rand.Int63(),
		inflight:       make(map[int]chan DataType),
	}

	go sess.handleListen()

	return sess, nil
}

func (s *Session) doRequest(data []byte, reqId int, c chan DataType) error {
	_, err := s.conn.WriteTo(data, s.addr)
	if err != nil {
		return err
	}

	s.inflight[reqId] = c
	go func() {
		<-time.After(500 * time.Millisecond)
		s.lock.Lock()
		defer s.lock.Unlock()

		if c, ok := s.inflight[reqId]; ok {
			// haven't received a response yet
			close(c)
			delete(s.inflight, reqId)
		}
	}()

	return nil
}

func (s *Session) handleListen() {
	b := make([]byte, 65500)

	for {
		n, err := s.conn.Read(b)

		if err != nil {
			continue
		}

		decoded, _, err := Decode(bytes.NewReader(b[:n]))
		if err != nil {
			continue
		}

		s.lock.Lock()
		reqId := int(decoded.(Sequence)[1].(Sequence)[0].(Int))

		switch decoded.(Sequence)[3].(type) {
		case String:
			encrypted := []byte(decoded.(Sequence)[3].(String))
			engineStuff, _, err := Decode(bytes.NewReader([]byte(decoded.(Sequence)[2].(String))))
			if err != nil {
				continue
			}

			s.engineBoots = int32(engineStuff.(Sequence)[1].(Int))
			s.engineTime = int32(engineStuff.(Sequence)[2].(Int))

			priv := []byte(engineStuff.(Sequence)[5].(String))

			result, _, err := Decode(bytes.NewReader(s.decrypt(encrypted, priv)))

			if err != nil {
				continue
			}

			responseData := result.(Sequence)[2]

			switch responseData.(type) {
			case GetResponse:
				reqId = int(responseData.(GetResponse)[0].(Int))

			case Report:
				reqId = int(responseData.(Report)[0].(Int))
			}
		}

		if c, ok := s.inflight[reqId]; ok {
			c <- decoded
			delete(s.inflight, reqId)
		}
		s.lock.Unlock()
	}

}

func (s *Session) Discover() error {
	reqId := int(rand.Intn(100000))

	encodedEngineData, err := Sequence{
		String(""),
		Int(0),
		Int(0),
		String(""),
		String(""),
		String(""),
	}.Encode()

	if err != nil {
		return err
	}

	discoverySequence, err := Sequence{
		Int(3),
		Sequence{
			Int(reqId),
			Int(65507),
			String("\x04"),
			Int(3),
		},
		String(encodedEngineData),
		Sequence{
			String(""),
			String(""),
			GetRequest{
				Int(0),
				Int(0),
				Int(0),
				Sequence{},
			},
		},
	}.Encode()

	if err != nil {
		return err
	}

	var decoded DataType
	var ok bool

	for i := 0; i < 3; i++ {
		c := make(chan DataType)
		s.doRequest(discoverySequence, int(reqId), c)

		decoded, ok = <-c
		if ok {
			break
		} else {
			if i == 2 {
				return errors.New("discovery failed")
			}
		}
	}

	engineStuff, _, err := Decode(bytes.NewReader([]byte(decoded.(Sequence)[2].(String))))
	if err != nil {
		return err
	}

	s.engineID = []byte(engineStuff.(Sequence)[0].(String))
	s.engineBoots = int32(engineStuff.(Sequence)[1].(Int))
	s.engineTime = int32(engineStuff.(Sequence)[2].(Int))

	s.privKey = passphraseToKey(s.privPassphrase, s.engineID)[:16]
	s.authKey = passphraseToKey(s.authPassphrase, s.engineID)

	return nil
}

func (s *Session) Get(oid []byte) (interface{}, error) {
	reqId := Int(rand.Int31())

	getReq, err := Sequence{
		String(s.engineID),
		String(""),
		GetRequest{
			reqId,
			Int(0),
			Int(0),
			Sequence{
				Sequence{
					ObjectIdentifier(oid),
					Null,
				},
			},
		},
	}.Encode()

	if err != nil {
		return nil, err
	}

	encrypted, priv := s.encrypt(getReq)

	packet, err := s.constructPacket(encrypted, priv)
	if err != nil {
		return nil, err
	}

	var decoded DataType
	var ok bool

	for i := 0; i < 3; i++ {
		c := make(chan DataType)
		s.doRequest(packet, int(reqId), c)

		decoded, ok = <-c
		if ok {
			break
		} else {
			if i == 2 {
				return nil, errors.New("timeout")
			}
		}
	}

	encrypted = []byte(decoded.(Sequence)[3].(String))
	engineStuff, _, err := Decode(bytes.NewReader([]byte(decoded.(Sequence)[2].(String))))

	if err != nil {
		return nil, err
	}

	s.engineBoots = int32(engineStuff.(Sequence)[1].(Int))
	s.engineTime = int32(engineStuff.(Sequence)[2].(Int))

	priv = []byte(engineStuff.(Sequence)[5].(String))

	result, _, err := Decode(bytes.NewReader(s.decrypt(encrypted, priv)))
	return result, err
}

func (s *Session) GetNext(oid []byte) (interface{}, error) {
	reqId := Int(rand.Int31())

	getNextReq, err := Sequence{
		String(s.engineID),
		String(""),
		GetNextRequest{
			reqId,
			Int(0),
			Int(0),
			Sequence{
				Sequence{
					ObjectIdentifier(oid),
					Null,
				},
			},
		},
	}.Encode()

	if err != nil {
		return nil, err
	}

	encrypted, priv := s.encrypt(getNextReq)

	packet, err := s.constructPacket(encrypted, priv)
	if err != nil {
		return nil, err
	}

	s.conn.WriteTo(packet, s.addr)

	b := make([]byte, 65500)
	n, err := s.conn.Read(b)
	if err != nil {
		log.Fatal(err)
	}

	decoded, _, err := Decode(bytes.NewReader(b[:n]))
	if err != nil {
		return nil, err
	}

	encrypted = []byte(decoded.(Sequence)[3].(String))
	engineStuff, _, err := Decode(bytes.NewReader([]byte(decoded.(Sequence)[2].(String))))
	if err != nil {
		return nil, err
	}

	s.engineBoots = int32(engineStuff.(Sequence)[1].(Int))
	s.engineTime = int32(engineStuff.(Sequence)[2].(Int))

	priv = []byte(engineStuff.(Sequence)[5].(String))

	result, _, err := Decode(bytes.NewReader(s.decrypt(encrypted, priv)))
	return result, err
}

func (s *Session) constructPacket(encrypted, priv []byte) ([]byte, error) {
	msgId := Int(rand.Int31())

	v3Header, err := Sequence{
		String(s.engineID),
		Int(s.engineBoots),
		Int(s.engineTime),
		String(s.user),
		String(strings.Repeat("\x00", 12)),
		String(priv),
	}.Encode()

	if err != nil {
		return nil, err
	}

	packet, err := Sequence{
		Int(3),
		Sequence{
			msgId,
			Int(65507),
			String("\x07"),
			Int(3),
		},
		String(v3Header),
		String(encrypted),
	}.Encode()

	if err != nil {
		return nil, err
	}

	authParam := s.auth(packet)

	return bytes.Replace(packet, bytes.Repeat([]byte{0}, 12), authParam, 1), nil
}

func (s *Session) Close() error {
	return s.conn.Close()
}
