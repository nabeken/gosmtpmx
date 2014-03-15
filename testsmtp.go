package gosmtpmx

import (
	"net"
)

func NewTestServer() error {
	l, e := net.Listen("tcp", "127.0.0.1:10025")
	if e != nil {
		return e
	}
	defer l.Close()

	rw, ae := l.Accept()
	if ae != nil {
		return ae
	}
	if _, err := rw.Write([]byte("220 mx.example.org ready\r\n")); err != nil {
		return err
	}
	if err := rw.Close(); err != nil {
		return err
	}
	return nil
}
