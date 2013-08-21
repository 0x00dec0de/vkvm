package vnc

import (
	"encoding/binary"
	"net"
)

type Conn struct {
	c *net.Conn
	s *net.Conn

	srv *Server
	cli *Client

	MessageCli chan *Message
	messagecli chan *Message

	MessageSrv chan *Message
	messagesrv chan *Message

	PixelFormat *PixelFormat

	ColorMap [256]Color

	Quit        chan bool
	Ready       bool
	Encs        []Encoding
	DesktopName string
	Exclusive   bool
}

func (srv *Server) newConn(c *net.Conn) *Conn {
	messagecli := make(chan *Message, srv.c.MaxMsg)
	messagesrv := make(chan *Message, srv.c.MaxMsg)
	quit := make(chan bool)

	defaultPixelFormat := &PixelFormat{
		BPP:        32,
		Depth:      24,
		BigEndian:  false,
		TrueColor:  true,
		RedMax:     255,
		GreenMax:   255,
		BlueMax:    255,
		RedShift:   16,
		GreenShift: 8,
		BlueShift:  0,
	}

	return &Conn{
		s:           c,
		c:           c,
		srv:         srv,
		PixelFormat: defaultPixelFormat,
		MessageCli:  messagecli,
		messagecli:  messagecli,
		MessageSrv:  messagesrv,
		messagesrv:  messagesrv,
		Quit:        quit,
		Ready:       false,
	}
}

func (cli *Client) newConn(c *net.Conn) *Conn {
	messagecli := make(chan *Message, cli.c.MaxMsg)
	messagesrv := make(chan *Message, cli.c.MaxMsg)
	quit := make(chan bool)

	return &Conn{
		c:           c,
		s:           c,
		cli:         cli,
		PixelFormat: &PixelFormat{},
		MessageCli:  messagecli,
		messagecli:  messagecli,
		MessageSrv:  messagesrv,
		messagesrv:  messagesrv,
		Quit:        quit,
		Ready:       false,
	}
}

func (c *Conn) Close() {
	if c.c != nil {
		nc := *c.c
		nc.Close()
		c.Quit <- true
	}
}

func (c *Conn) readByte() (b byte, err error) {
	err = binary.Read(*c.c, binary.BigEndian, &b)
	return
}
