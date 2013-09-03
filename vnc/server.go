package vnc

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
)

type Server struct {
	c     *ServerConfig
	conns chan *Conn
	Conns <-chan *Conn
}

type ServerConfig struct {
	Version string

	AuthTypes []AuthType
	MaxConn   int
	MaxMsg    int

	Width  int
	Height int

	Messages []Message
}

func NewServer(cfg *ServerConfig) *Server {
	if cfg.Version == "" {
		cfg.Version = "RFB 003.008\n"
	}
	if cfg.Width < 1 {
		cfg.Width = 720
	}
	if cfg.Height < 1 {
		cfg.Height = 400
	}
	if cfg.AuthTypes == nil {
		cfg.AuthTypes = []AuthType{new(AuthTypeNone)}
	}
	conns := make(chan *Conn, cfg.MaxConn)
	return &Server{
		c:     cfg,
		conns: conns,
		Conns: conns,
	}
}

func (srv *Server) Serve(l net.Listener) error {
	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}
		conn := srv.newConn(&c)

		if err := conn.serverVersionHandshake(); err != nil {
			return err
		}
		if err := conn.serverSecurityHandshake(); err != nil {
			return err
		}
		if err := conn.serverInit(); err != nil {
			return err
		}
		select {
		case srv.conns <- conn:
		default:
		}
		go conn.serverServe()
	}
}

func (c *Conn) serverServe() {
	defer c.Close()

	typeMap := make(map[uint8]Message)
	defaultMessages := []Message{
		new(SetPixelFormatMsg),
		new(SetEncodingsMsg),
		new(FramebufferUpdateRequestMsg),
		new(KeyEventMsg),
		new(PointerEventMsg),
		new(ClientCutTextMsg),
	}

	for _, msg := range defaultMessages {
		typeMap[msg.Type()] = msg
	}

	if c.srv.c.Messages != nil {
		for _, msg := range c.srv.c.Messages {
			typeMap[msg.Type()] = msg
		}
	}

	go func() {
		for {
			messageType, err := c.readByte()
			if err != nil {
				fmt.Printf("server<-client: Error reading message type\n")
				return
			}
			msg, ok := typeMap[messageType]
			if !ok {
				fmt.Printf("Unsupported message type: %d\n", messageType)
				return
			}
			parsedMsg, err := msg.Read(c, *c.c)
			if err != nil {
				fmt.Printf("server<-client: er: %T %s\n", msg, err.Error())
				return
			} else {
				fmt.Printf("server<-client: ok: %T %+v\n", msg, parsedMsg)
			}
			c.MessageCli <- &parsedMsg
		}
	}()
	for {
		select {
		case msg := <-c.MessageSrv:
			m := *msg
			err := m.Write(c, *c.c)
			if err != nil {
				fmt.Printf("server->client: er: %T %s\n", msg, err.Error())
				return
			} else {
				fmt.Printf("server->client: ok: %T %+v\n", m, m)
			}
		}
	}
}

func (c *Conn) serverVersionHandshake() error {
	var protocolVersion [12]byte
	bw := bufio.NewWriter(*c.c)
	br := bufio.NewReader(*c.c)

	// Respond with the version we will support
	if _, err := bw.WriteString(c.srv.c.Version); err != nil {
		return err
	}
	bw.Flush()

	// 7.1.1, read the ProtocolVersion message sent by the server.
	if _, err := io.ReadFull(br, protocolVersion[:]); err != nil {
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
	c.MinorVersion = maxMinor
	c.MajorVersion = maxMajor

	return nil
}

func (c *Conn) serverSecurityHandshake() error {
	bw := bufio.NewWriter(*c.c)
	serverSecurityTypes := c.srv.c.AuthTypes
	var securityType uint8

	if c.MinorVersion >= 7 {
		var sectypes []uint8
		sectypes = []uint8{uint8(len(serverSecurityTypes))}
		for _, curAuth := range serverSecurityTypes {
			sectypes = append(sectypes, curAuth.Type())
		}
		if err := binary.Write(*c.c, binary.BigEndian, sectypes); err != nil {
			return err
		}

		if err := binary.Read(*c.c, binary.BigEndian, &securityType); err != nil {
			return err
		}
	} else {
		if err := binary.Write(*c.c, binary.BigEndian, uint32(authVNC)); err != nil {
			return err
		}
		securityType = authVNC
	}

	var authType AuthType
FindAuth:
	for _, curAuth := range serverSecurityTypes {
		if curAuth.Type() == securityType {
			authType = curAuth
			break FindAuth
		}
	}

	if authType == nil {
		return errors.New("no suitable auth schemes found. server supported")
	}

	if err := authType.Handler(c, *c.c); err != nil {
		e := err
		if err = binary.Write(*c.c, binary.BigEndian, uint32(1)); err != nil {
			return err
		}
		if c.MinorVersion >= 8 {
			reasonLen := uint32(len(e.Error()))
			if err = binary.Write(bw, binary.BigEndian, reasonLen); err != nil {
				return err
			}

			reason := []byte(e.Error())
			if err = binary.Write(bw, binary.BigEndian, &reason); err != nil {
				return err
			}
			bw.Flush()
		}
		return e
	}
	if err := binary.Write(*c.c, binary.BigEndian, uint32(0)); err != nil {
		return err
	}
	return nil
}

func (c *Conn) serverInit() error {
	bw := bufio.NewWriter(*c.c)
	var err error
	var sharedFlag uint8
	if err = binary.Read(*c.c, binary.BigEndian, &sharedFlag); err != nil {
		return err
	}
	_ = sharedFlag

	if err = binary.Write(bw, binary.BigEndian, uint16(c.srv.c.Width)); err != nil {
		return err
	}

	if err = binary.Write(bw, binary.BigEndian, uint16(c.srv.c.Height)); err != nil {
		return err
	}

	var format []byte
	if format, err = writePixelFormat(c.PixelFormat); err != nil {
		return err
	}
	if err = binary.Write(bw, binary.BigEndian, format); err != nil {
		return err
	}

	padding := []uint8{0, 0, 0}
	if err = binary.Write(bw, binary.BigEndian, padding); err != nil {
		return err
	}

	nameBytes := []uint8(c.DesktopName)
	nameLen := uint32(cap(nameBytes))
	if err = binary.Write(bw, binary.BigEndian, nameLen); err != nil {
		return err
	}

	if err = binary.Write(bw, binary.BigEndian, nameBytes); err != nil {
		return err
	}

	bw.Flush()
	return nil
}
