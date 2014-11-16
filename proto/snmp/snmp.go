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
	log.Println("doRequest", reqId)
	_, err := s.conn.WriteTo(data, s.addr)
	if err != nil {
		return err
	}

	s.inflight[reqId] = c
	go func() {
		<-time.After(100 * time.Millisecond)
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

		decoded, _ := Decode(bytes.NewReader(b[:n]))

		s.lock.Lock()
		reqId := int(decoded.(Sequence)[1].(Sequence)[0].(Int))

		switch decoded.(Sequence)[3].(type) {
		case String:
			encrypted := []byte(decoded.(Sequence)[3].(String))
			engineStuff, _ := Decode(bytes.NewReader([]byte(decoded.(Sequence)[2].(String))))

			s.engineBoots = int32(engineStuff.(Sequence)[1].(Int))
			s.engineTime = int32(engineStuff.(Sequence)[2].(Int))

			priv := []byte(engineStuff.(Sequence)[5].(String))

			result, _ := Decode(bytes.NewReader(s.decrypt(encrypted, priv)))
			reqId = int(result.(Sequence)[2].(GetResponse)[0].(Int))
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

	discoverySequence := Sequence{
		Int(3),
		Sequence{
			Int(reqId),
			Int(65507),
			String("\x04"),
			Int(3),
		},
		String(Sequence{
			String(""),
			Int(0),
			Int(0),
			String(""),
			String(""),
			String(""),
		}.Encode()),
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

	c := make(chan DataType)
	err := s.doRequest(discoverySequence, reqId, c)

	if err != nil {
		return err
	}

	decoded, ok := <-c
	if !ok {
		return errors.New("discovery failed")
	}

	log.Printf("Request ID: %#+v", decoded.(Sequence)[1].(Sequence)[0].(Int))

	engineStuff, _ := Decode(bytes.NewReader([]byte(decoded.(Sequence)[2].(String))))

	s.engineID = []byte(engineStuff.(Sequence)[0].(String))
	s.engineBoots = int32(engineStuff.(Sequence)[1].(Int))
	s.engineTime = int32(engineStuff.(Sequence)[2].(Int))

	s.privKey = passphraseToKey(s.privPassphrase, s.engineID)[:16]
	s.authKey = passphraseToKey(s.authPassphrase, s.engineID)

	return nil
}

func (s *Session) Get(oid []byte) interface{} {
	reqId := Int(rand.Int31())

	getReq := Sequence{
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

	encrypted, priv := s.encrypt(getReq)

	packet := s.constructPacket(encrypted, priv)

	c := make(chan DataType)
	s.doRequest(packet, int(reqId), c)

	decoded, ok := <-c
	if !ok {
		return nil
	}

	encrypted = []byte(decoded.(Sequence)[3].(String))
	engineStuff, _ := Decode(bytes.NewReader([]byte(decoded.(Sequence)[2].(String))))

	s.engineBoots = int32(engineStuff.(Sequence)[1].(Int))
	s.engineTime = int32(engineStuff.(Sequence)[2].(Int))

	priv = []byte(engineStuff.(Sequence)[5].(String))

	result, _ := Decode(bytes.NewReader(s.decrypt(encrypted, priv)))
	return result
}

func (s *Session) GetNext(oid []byte) interface{} {
	reqId := Int(rand.Int31())

	getNextReq := Sequence{
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

	encrypted, priv := s.encrypt(getNextReq)

	packet := s.constructPacket(encrypted, priv)

	s.conn.WriteTo(packet, s.addr)

	b := make([]byte, 65500)
	n, err := s.conn.Read(b)
	if err != nil {
		log.Fatal(err)
	}

	decoded, _ := Decode(bytes.NewReader(b[:n]))

	encrypted = []byte(decoded.(Sequence)[3].(String))
	engineStuff, _ := Decode(bytes.NewReader([]byte(decoded.(Sequence)[2].(String))))

	s.engineBoots = int32(engineStuff.(Sequence)[1].(Int))
	s.engineTime = int32(engineStuff.(Sequence)[2].(Int))

	priv = []byte(engineStuff.(Sequence)[5].(String))

	result, _ := Decode(bytes.NewReader(s.decrypt(encrypted, priv)))
	return result
}

func (s *Session) constructPacket(encrypted, priv []byte) []byte {
	msgId := Int(rand.Int31())

	v3Header := Sequence{
		String(s.engineID),
		Int(s.engineBoots),
		Int(s.engineTime),
		String(s.user),
		String(strings.Repeat("\x00", 12)),
		String(priv),
	}.Encode()

	packet := Sequence{
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

	authParam := s.auth(packet)

	return bytes.Replace(packet, bytes.Repeat([]byte{0}, 12), authParam, 1)
}

func (s *Session) Close() error {
	return s.conn.Close()
}
