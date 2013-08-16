package vnc

import (
	"bufio"
	"encoding/binary"
	"net"
)

type Conn struct {
	c *net.Conn
	s *net.Conn

	srv *Server
	cli *Client
	br  *bufio.Reader
	bw  *bufio.Writer

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
		br:         bufio.NewReader(c),
		bw:         bufio.NewWriter(c),
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
		br:         bufio.NewReader(*c),
		bw:         bufio.NewWriter(*c),
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
	err = binary.Read(c.br, binary.BigEndian, &b)
	return
}
