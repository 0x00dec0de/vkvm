package main

import (
	"bufio"
	"encoding/binary"
	"io"
	"log"
	"net"
)

type Conn struct {
	sc net.Conn
	cc net.Conn

	Major uint8
	Minor uint8

	Width  uint16
	Height uint16

	SharedFlag uint8

	Name string

	PixelFormat PixelFormat

	br *bufio.Reader
	bw *bufio.Writer

	err error

	scHostPort string
	ccHostPort string
	Password   string

	MsgChan  chan byte
	MsgDone  chan bool
	InitDone chan bool
}

func (c *Conn) Read(b []byte) (n int, err error) {
	if b, err = c.br.Peek(1); err != nil {
		return 1, err
	}
	if c.sc != nil {
		switch b[0] {
		case 0:
			buf := make([]byte, 20)
			n, err = io.ReadFull(c.br, buf)
			b = buf
			return
		case 2:
			var nEncs uint16
			if b, err = c.br.Peek(4); err != nil {
				return
			}
			tt := binary.BigEndian.Uint16(b[2:3])
			log.Printf("%+v\n", tt)
			nEncs = tt
			buf := make([]byte, 4+4*nEncs)
			n, err = io.ReadFull(c.br, buf)
			b = buf
			return
		case 3:
			buf := make([]byte, 10)
			n, err = io.ReadFull(c.br, buf)
			b = buf
			return
		case 4:
			buf := make([]byte, 8)
			n, err = io.ReadFull(c.br, buf)
			b = buf
			return
		case 5:
			buf := make([]byte, 6)
			n, err = io.ReadFull(c.br, buf)
			b = buf
			return
		case 6:
			if b, err = c.br.Peek(8); err != nil {
				return
			}
			var Len uint32
			Len = binary.BigEndian.Uint32(b[4:8])
			buf := make([]byte, 8+Len)
			n, err = io.ReadFull(c.br, buf)
			b = buf
			return
		}
	}
	if c.cc != nil {
		switch b[0] {
		case 0:
			var nRecs uint16
			if b, err = c.br.Peek(4); err != nil {
				return
			}
			nRecs = binary.BigEndian.Uint16(b[2:3])
			buf := make([]byte, 4+nRecs*12)
			n, err = io.ReadFull(c.br, buf)
			b = buf
			return
		case 1:
			var nColors uint16
			if b, err = c.br.Peek(6); err != nil {
				return
			}
			nColors = binary.BigEndian.Uint16(b[4:5])
			buf := make([]byte, 6+nColors*6)
			n, err = io.ReadFull(c.br, buf)
			b = buf
			return
		case 2:
			buf := make([]byte, 1)
			n, err = io.ReadFull(c.br, buf)
			b = buf
			return
		case 3:
			var Len uint32
			if b, err = c.br.Peek(6); err != nil {
				return
			}
			Len = binary.BigEndian.Uint32(b[4:8])
			buf := make([]byte, 8+Len)
			n, err = io.ReadFull(c.br, buf)
			b = buf
			return
		}
	}
	return
}

func (c *Conn) Write(b []byte) (n int, err error) {
	n, err = c.bw.Write(b)
	c.bw.Flush()
	return
}

func (c *Conn) Close() error {
	if c.sc != nil {
		c.sc.Close()
	}
	if c.cc != nil {
		c.cc.Close()
	}
	return nil
}
