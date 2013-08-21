package main

import (
	"./vnc"
	"crypto/des"
	"crypto/rand"
	"encoding/binary"
	"io"
	"net"
)

type ServerAuthTypeVNC byte

func (*ServerAuthTypeVNC) Type() uint8 {
	return 2
}

func (*ServerAuthTypeVNC) Handler(srv *vnc.Conn, rw io.ReadWriter) (err error) {
	challenge := make([]uint8, 16)
	response := make([]uint8, 16)
	_, err = rand.Read(challenge)
	if err != nil {
		return err
	}
	if err := binary.Write(rw, binary.BigEndian, challenge); err != nil {
		return err
	}
	if err := binary.Read(rw, binary.BigEndian, &response); err != nil {
		return err
	}

	// doing external auth

	cli := vnc.NewClient(&vnc.ClientConfig{AuthTypes: []vnc.AuthType{new(ClientAuthTypeVNC)}})

	n, err := net.Dial("tcp", "127.0.0.1:5900")
	if err != nil {
		return err
	}
	var conn *vnc.Conn
	if conn, err = cli.Serve(n); err != nil {
		return err
	}

	p.Lock()
	srv.DesktopName = conn.DesktopName
	rConn := &rConn{c: conn, password: []byte("njkcnjd")}
	p.Targets[srv] = rConn
	p.Unlock()

	return nil
}

type ClientAuthTypeVNC byte

func (*ClientAuthTypeVNC) Type() uint8 {
	return 2
}

func (*ClientAuthTypeVNC) Handler(c *vnc.Conn, rw io.ReadWriter) (err error) {
	challenge := make([]uint8, 16)

	if err := binary.Read(rw, binary.BigEndian, &challenge); err != nil {
		return err
	}

	//external auth
	pwd := []byte("njkcnjd")
	if len(pwd) > 8 {
		pwd = pwd[:8]
	}
	if len(pwd) < 8 {
		if x := len(pwd); x < 8 {
			for i := 8 - x; i > 0; i-- {
				pwd = append(pwd, byte(0))
			}
		}
	}

	newpwd := make([]byte, 8)
	for i := 0; i < 8; i++ {
		c := pwd[i]
		c = ((c & 0x01) << 7) + ((c & 0x02) << 5) + ((c & 0x04) << 3) + ((c & 0x08) << 1) +
			((c & 0x10) >> 1) + ((c & 0x20) >> 3) + ((c & 0x40) >> 5) + ((c & 0x80) >> 7)
		newpwd[i] = c
	}

	enc, err := des.NewCipher(newpwd)
	if err != nil {
		return err
	}
	response := make([]byte, 16)

	enc.Encrypt(response[:8], challenge[:8])
	enc.Encrypt(response[8:], challenge[8:])
	if err = binary.Write(rw, binary.BigEndian, response); err != nil {
		return err
	}

	return nil
}
