package main

import (
	"./vnc"
	"crypto/des"
	"crypto/rand"
	"encoding/binary"
	"io"
)

type ServerAuthTypeVNC Proxy

func (p *ServerAuthTypeVNC) Type() uint8 {
	return 2
}

func (p *ServerAuthTypeVNC) Handler(c *vnc.Conn, rw io.ReadWriter) (err error) {
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

	r, err := vnc.Client("127.0.0.1:5900", []byte("njkcnjd"))
	rconn := &rConn{HostPort: "127.0.0.1:5900", Password: []byte("njkcnjd")}
	err = p.Handler(rconn, rw)
	if err != nil {
		return err
	}

	p.Lock()
	p.Targets[c] = rconn
	p.Unlock()

	return nil
}

type ClientAuthTypeVNC byte

func (p *ClientAuthTypeVNC) Type() uint8 {
	return 2
}

func (p *ClientAuthTypeVNC) Handler(c *vnc.Conn, rw io.ReadWriter) (err error) {
	challenge := make([]uint8, 16)

	if err := binary.Read(rw, binary.BigEndian, &challenge); err != nil {
		return err
	}
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
