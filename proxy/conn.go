package main

import (
	"bufio"
	"io"
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
		return
	}
	switch b[0] {
	case 0:
		if len(b) < 20 {
			b = make([]byte, 20)
		}
		n, err = io.ReadFull(c.br, b)
		return
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
