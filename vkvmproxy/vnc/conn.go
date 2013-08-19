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

	MessageCli <-chan *Message
	messagecli chan *Message

	MessageSrv chan<- *Message
	messagesrv chan *Message

	PixelFormat *PixelFormat

	ColorMap [256]Color

	Encs        []Encoding
	DesktopName string
	Exclusive   bool
}

func (srv *Server) newConn(c net.Conn) *Conn {
	messagecli := make(chan *Message, srv.c.MaxMsg)
	messagesrv := make(chan *Message, srv.c.MaxMsg)
	return &Conn{
		s:          &c,
		srv:        srv,
		MessageCli: messagecli,
		messagecli: messagecli,
		MessageSrv: messagesrv,
		messagesrv: messagesrv,
	}
}

func (cli *Client) newConn(c *net.Conn) *Conn {
	messagecli := make(chan *Message, cli.c.MaxMsg)
	messagesrv := make(chan *Message, cli.c.MaxMsg)
	return &Conn{
		s:          c,
		cli:        cli,
		MessageCli: messagecli,
		messagecli: messagecli,
		MessageSrv: messagesrv,
		messagesrv: messagesrv,
	}
}

func (c *Conn) Close() {
	if c.s != nil {
		//		c.s.Close()
	}
	if c.c != nil {
		//	c.c.Close()
	}
}

func (c *Conn) readByte() (b byte, err error) {
	err = binary.Read(*c.c, binary.BigEndian, &b)
	return
}
