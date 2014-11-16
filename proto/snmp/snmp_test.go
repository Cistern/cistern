package snmp

import (
	"testing"
)

func TestSession(t *testing.T) {
	sess, err := NewSession("10.2.33.100:161", "adminusr", "snmpPASSWORD", "encryptionKEY")
	if err != nil {
		t.Fatal(err)
	}

	sess.Discover()

	t.Log(sess.Get([]byte{0x2b, 0x06, 0x01, 0x02, 0x01, 0x02, 0x02, 0x01, 0x10, 0x01}).(Sequence)[2].(GetResponse)[3].(Sequence)[0].(Sequence)[1])
	t.Log(sess.Get([]byte{0x2b, 0x06, 0x01, 0x02, 0x01, 0x01, 0x01, 0x00}).(Sequence)[2].(GetResponse)[3].(Sequence)[0].(Sequence)[1])
	t.Log(sess.Get([]byte{0x2b, 6, 1, 2, 1, 2, 2, 1, 2, 1}).(Sequence)[2].(GetResponse)[3].(Sequence)[0].(Sequence)[1])
	t.Log(sess.GetNext([]byte{0x2b, 6, 1, 2, 1, 4, 20, 1, 1}).(Sequence)[2].(GetResponse)[3])
	t.Log(sess.Get([]byte{0x2b, 6, 1, 2, 1, 2, 2, 1, 2, 2}).(Sequence)[2].(GetResponse)[3].(Sequence)[0].(Sequence)[1])
	t.Log(sess.Get([]byte{0x2b, 6, 1, 2, 1, 2, 2, 1, 2, 2}).(Sequence)[2].(GetResponse)[3].(Sequence)[0].(Sequence)[1])
	t.Log(sess.Get([]byte{0x2b, 6, 1, 2, 1, 2, 2, 1, 2, 2}).(Sequence)[2].(GetResponse)[3].(Sequence)[0].(Sequence)[1])
	t.Log(sess.Get([]byte{0x2b, 6, 1, 2, 1, 2, 2, 1, 2, 2}).(Sequence)[2].(GetResponse)[3].(Sequence)[0].(Sequence)[1])

	err = sess.Close()
	if err != nil {
		t.Fatal(err)
	}

}
