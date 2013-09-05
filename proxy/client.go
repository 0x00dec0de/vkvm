package main

import (
	"bufio"
	"crypto/des"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
)

type Client struct {
	conns chan *Conn
	Conns <-chan *Conn
}

func NewClient() *Client {
	conns := make(chan *Conn, 1024)
	return &Client{
		conns: conns,
		Conns: conns,
	}
}

func (cli *Client) Serve(c net.Conn) (*Conn, error) {
	var err error
	conn := cli.newConn(c)
	log.Printf("clientVersionHandshake\n")
	if err = conn.clientVersionHandshake(); err != nil {
		return nil, err
	}
	log.Printf("clientSecurityHandshake\n")
	if err = conn.clientSecurityHandshake(); err != nil {
		return nil, err
	}
	log.Printf("clientInit\n")
	if err = conn.clientInit(); err != nil {
		return nil, err
	}
	log.Printf("clientServe\n")
	go conn.clientServe()
	return conn, err
}

func (cli *Client) newConn(c net.Conn) *Conn {
	return &Conn{
		cc:       c,
		br:       bufio.NewReader(c),
		bw:       bufio.NewWriter(c),
		MsgChan:  make(chan []byte, 0),
		MsgDone:  make(chan bool, 0),
		InitDone: make(chan bool, 1),
	}
}

func (c *Conn) clientVersionHandshake() error {
	var protocolVersion [12]byte

	if err := binary.Read(c.cc, binary.BigEndian, &protocolVersion); err != nil {
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

	if maxMinor < 8 {
		return fmt.Errorf("unsupported minor version, less than 8: %d", maxMinor)
	}

	if err = binary.Write(c.cc, binary.BigEndian, []byte("RFB 003.008\n")); err != nil {
		return err
	}
	c.bw.Flush()
	return nil
}

func (c *Conn) clientSecurityHandshake() error {
	var err error
	var numSecurityTypes uint8
	if err = binary.Read(c.cc, binary.BigEndian, &numSecurityTypes); err != nil {
		return err
	}

	if numSecurityTypes == 0 {
		var reasonLength uint32
		if err = binary.Read(c.cc, binary.BigEndian, &reasonLength); err != nil {
			return err
		}
		reasonText := make([]byte, reasonLength)
		if err = binary.Read(c.cc, binary.BigEndian, &reasonText); err != nil {
			return err
		}
		return fmt.Errorf("no security types: %s", reasonText)
	}

	securityTypes := make([]uint8, numSecurityTypes)
	if err = binary.Read(c.cc, binary.BigEndian, &securityTypes); err != nil {
		return err
	}

	auth := false
	for _, t := range securityTypes {
		if t == uint8(2) {
			auth = true
			break
		}
	}

	if !auth {
		return fmt.Errorf("no suitable auth schemes found.")
	}

	// Respond back with the security type we'll use
	if err = binary.Write(c.cc, binary.BigEndian, uint8(2)); err != nil {
		return err
	}

	if err = c.clientAuth(); err != nil {
		return err
	}

	// 7.1.3 SecurityResult Handshake
	var securityResult uint32
	if err = binary.Read(c.cc, binary.BigEndian, &securityResult); err != nil {
		return err
	}

	if securityResult == 1 {
		var reasonLength uint32
		if err = binary.Read(c.cc, binary.BigEndian, &reasonLength); err != nil {
			return err
		}
		reasonText := make([]byte, reasonLength)
		if err = binary.Read(c.cc, binary.BigEndian, &reasonText); err != nil {
			return err
		}
		return fmt.Errorf("security handshake failed: %s", reasonText)
	}
	return nil
}

func (c *Conn) clientInit() error {
	var err error

	if err = binary.Write(c.cc, binary.BigEndian, c.SharedFlag); err != nil {
		return err
	}

	if err = binary.Read(c.cc, binary.BigEndian, &c.Width); err != nil {
		return err
	}
	if err = binary.Read(c.cc, binary.BigEndian, &c.Height); err != nil {
		return err
	}

	if err = binary.Read(c.cc, binary.BigEndian, &c.PixelFormat); err != nil {
		return err
	}
	var nameLength uint32
	if err = binary.Read(c.cc, binary.BigEndian, &nameLength); err != nil {
		return err
	}

	nameBytes := make([]uint8, nameLength)
	if err = binary.Read(c.cc, binary.BigEndian, &nameBytes); err != nil {
		return err
	}

	c.Name = string(nameBytes)
	c.InitDone <- true
	return nil
}

func (c *Conn) clientServe() {
	go func() {
		defer c.Close()
		for {
			buf, err := c.Read()
			if err == io.EOF {
				continue
			}
			if err != nil {
				fmt.Printf("client<-server: Error reading message type, %s\n", err.Error())
				return
			}
			log.Printf("cc: start send to chan\n")
			c.MsgChan <- buf
			log.Printf("cc: stop send to chan\n")
			log.Printf("cc: start wait for done\n")
			<-c.MsgDone
			log.Printf("cc: stop wait for done\n")
		}
	}()
}

func (c *Conn) clientAuth() (err error) {
	challenge := make([]uint8, 16)

	if err := binary.Read(c.cc, binary.BigEndian, &challenge); err != nil {
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
	if err = binary.Write(c.cc, binary.BigEndian, response); err != nil {
		return err
	}
	return nil
}
