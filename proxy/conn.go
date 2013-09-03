package main

import (
	"bufio"
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
	log.Printf("READ\n")
	if b, err = c.br.Peek(1); err != nil {
		return
	}
	log.Printf("%+v\n", b)
	return
}

func (c *Conn) Write(b []byte) (int, error) {
	log.Printf("WRITE\n")
	return 0, nil
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
