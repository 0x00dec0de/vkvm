package main

import (
	"bufio"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
)

func NewServer() *Server {
	conns := make(chan *Conn, 1024)
	return &Server{
		conns: conns,
		Conns: conns,
	}
}

type Server struct {
	conns chan *Conn
	Conns <-chan *Conn
}

func (srv *Server) Serve(l net.Listener) {
	for {
		c, err := l.Accept()
		if err != nil {
			return
		}
		conn := srv.newConn(c)
		log.Printf("serverVersionHandshake\n")
		if err := conn.serverVersionHandshake(); err != nil {
			c.Close()
		}
		log.Printf("serverSecurityHandshake\n")
		if err := conn.serverSecurityHandshake(); err != nil {
			c.Close()
		}
		log.Printf("serverInit\n")
		if err := conn.serverInit(); err != nil {
			c.Close()
		}
		select {
		case srv.conns <- conn:
		default:
		}
		log.Printf("serverServe\n")
		go conn.serverServe()
	}
}

func (srv *Server) newConn(c net.Conn) *Conn {
	return &Conn{
		sc:       c,
		br:       bufio.NewReader(c),
		bw:       bufio.NewWriter(c),
		MsgChan:  make(chan byte, 1),
		MsgDone:  make(chan bool, 1),
		InitDone: make(chan bool, 1),
	}
}

func (c *Conn) serverVersionHandshake() error {
	var protocolVersion [12]byte

	// Respond with the version we will support
	if _, err := c.bw.WriteString("RFB 003.008\n"); err != nil {
		return err
	}
	c.bw.Flush()

	// 7.1.1, read the ProtocolVersion message sent by the server.
	if _, err := io.ReadFull(c.br, protocolVersion[:]); err != nil {
		return err
	}

	var maxMajor, maxMinor uint8
	_, err := fmt.Sscanf(string(protocolVersion[:]), "RFB %d.%d\n", &maxMajor, &maxMinor)
	if err != nil {
		return err
	}

	if maxMajor < 3 {
		return fmt.Errorf("unsupported major version, less than 3: %d", maxMajor)
	}

	if maxMinor < 3 {
		return fmt.Errorf("unsupported minor version, less than 3: %d", maxMinor)
	}

	c.Minor = maxMinor
	c.Major = maxMajor
	return nil
}

func (c *Conn) serverSecurityHandshake() error {
	var securityType uint8

	if c.Minor >= 7 {
		if err := binary.Write(c.bw, binary.BigEndian, []uint8{uint8(1), uint8(2)}); err != nil {
			return err
		}
		c.bw.Flush()

		if err := binary.Read(c.br, binary.BigEndian, &securityType); err != nil {
			return err
		}
	} else {
		if err := binary.Write(c.bw, binary.BigEndian, uint32(2)); err != nil {
			return err
		}
		securityType = 2
	}

	if err := c.serverAuth(); err != nil {
		e := err
		if err = binary.Write(c.bw, binary.BigEndian, uint32(1)); err != nil {
			return err
		}
		c.bw.Flush()
		if c.Minor >= 8 {
			reasonLen := uint32(len(e.Error()))
			if err = binary.Write(c.bw, binary.BigEndian, reasonLen); err != nil {
				return err
			}

			reason := []byte(e.Error())
			if err = binary.Write(c.bw, binary.BigEndian, &reason); err != nil {
				return err
			}
			c.bw.Flush()
		}
		return e
	}
	if err := binary.Write(c.bw, binary.BigEndian, uint32(0)); err != nil {
		return err
	}
	c.bw.Flush()
	return nil
}

func (c *Conn) serverServe() {
	go func() {
		defer c.Close()
		for {
			msg, err := c.br.Peek(1)
			if err == io.EOF {
				continue
			}
			if err != nil {
				fmt.Printf("server<-client: Error reading message type\n")
				return
			}
			c.MsgChan <- msg[0]
			<-c.MsgDone
		}
	}()
}

func (c *Conn) serverAuth() (err error) {
	challenge := make([]uint8, 16)
	response := make([]uint8, 16)
	_, err = rand.Read(challenge)
	if err != nil {
		return err
	}
	if err := binary.Write(c.bw, binary.BigEndian, challenge); err != nil {
		return err
	}
	c.bw.Flush()
	if err := binary.Read(c.br, binary.BigEndian, &response); err != nil {
		return err
	}

	// doing external auth

	cli := NewClient()

	n, err := net.Dial("tcp", "127.0.0.1:5900")
	if err != nil {
		return err
	}
	var conn *Conn
	if conn, err = cli.Serve(n); err != nil {
		return err
	}

	p.Lock()
	p.Targets[c] = conn
	p.Unlock()
	return nil
}

func (c *Conn) serverInit() error {
	var cc *Conn
	var ok bool
	var err error
	var sharedFlag uint8
	if err = binary.Read(c.br, binary.BigEndian, &sharedFlag); err != nil {
		return err
	}
	_ = sharedFlag
	p.Lock()
	if cc, ok = p.Targets[c]; !ok {
		p.Unlock()
		return fmt.Errorf("failed to get client")
	} else {
		<-cc.InitDone
	}
	p.Unlock()
	log.Printf("%+v\n", cc)
	if err = binary.Write(c.bw, binary.BigEndian, cc.Width); err != nil {
		return err
	}
	log.Printf("Avail: %d Buffer: %d\n", c.bw.Available(), c.bw.Buffered())
	if err = binary.Write(c.bw, binary.BigEndian, cc.Height); err != nil {
		return err
	}
	log.Printf("Avail: %d Buffer: %d\n", c.bw.Available(), c.bw.Buffered())
	if err = binary.Write(c.bw, binary.BigEndian, cc.PixelFormat); err != nil {
		return err
	}
	log.Printf("Avail: %d Buffer: %d\n", c.bw.Available(), c.bw.Buffered())
	nameBytes := []uint8(cc.Name)
	nameLen := uint32(len(nameBytes))
	log.Printf("%d\n", nameLen)
	if err = binary.Write(c.bw, binary.BigEndian, nameLen); err != nil {
		return err
	}
	log.Printf("%s\n", nameBytes)
	if err = binary.Write(c.bw, binary.BigEndian, nameBytes); err != nil {
		return err
	}
	log.Printf("Avail: %d Buffer: %d\n", c.bw.Available(), c.bw.Buffered())
	c.bw.Flush()
	return nil
}
