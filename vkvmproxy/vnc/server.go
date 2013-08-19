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
	Width   int
	Height  int

	AuthTypes []AuthType
	MaxConn   int
	MaxMsg    int

	Messages []Message
}

func NewServer(cfg *ServerConfig) *Server {
	if cfg.Version == "" {
		cfg.Version = "RFB 003.008\n"
	}
	if cfg.Width < 1 {
		cfg.Width = 1
	}
	if cfg.Height < 1 {
		cfg.Height = 1
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

func (s *Server) Serve(l net.Listener) error {
	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}
		conn := s.newConn(c)
		if err := conn.versionHandshake(); err != nil {
			return err
		}
		if err := conn.securityHandshake(); err != nil {
			return err
		}
		//		if err := conn.serverInit(); err != nil {
		//		return err
		//}
		select {
		case s.conns <- conn:
		default:
		}
		go conn.serve()
	}
	panic("something wrong")
}

func (c *Conn) serve() {
	//	var err error
	defer c.Close()

	typeMap := make(map[uint8]Message)
	defaultMessages := []Message{
	//		new(SetPixelFormatMessage),
	//	new(SetEncodingMessage),
	//new(FrameBufferUpdateRequestMessage),
	//new(KeyEventMessage),
	//new(PointerEventMessage),
	//new(ClientCutTextMessage),
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
				break
			}
			msg, ok := typeMap[messageType]
			if !ok {
				// Unsupported message type! Bad!
				break
			}
			parsedMsg, err := msg.Read(c, *c.c)
			if err != nil {
				break
			}

			c.MessageSrv <- &parsedMsg
		}
	}()

}

func (c *Conn) versionHandshake() error {
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
		return fmt.Errorf("unsupported major version, less than 3: %d",
			maxMajor)
	}

	if maxMinor < 8 {
		return fmt.Errorf("unsupported minor version, less than 8: %d",
			maxMinor)
	}
	return nil
}

func (c *Conn) securityHandshake() error {
	bw := bufio.NewWriter(*c.c)
	serverSecurityTypes := c.srv.c.AuthTypes

	var sectypes []uint8
	sectypes = []uint8{uint8(len(serverSecurityTypes))}
	for _, curAuth := range serverSecurityTypes {
		sectypes = append(sectypes, curAuth.Type())
	}
	if err := binary.Write(*c.c, binary.BigEndian, sectypes); err != nil {
		return err
	}

	var securityType uint8
	if err := binary.Read(*c.c, binary.BigEndian, &securityType); err != nil {
		return err
	}

	var authType AuthType
FindAuth:
	for _, curAuth := range serverSecurityTypes {
		if curAuth.Type() == securityType {
			// We use the first matching supported authentication
			authType = curAuth
			break FindAuth
		}
	}

	if authType == nil {
		return errors.New("no suitable auth schemes found. server supported")
	}

	if err := authType.Handler(c, *c.c); err != nil {
		if err = binary.Write(*c.c, binary.BigEndian, uint32(1)); err != nil {
			return err
		}

		reasonLen := uint32(len(err.Error()))
		if err = binary.Write(bw, binary.BigEndian, reasonLen); err != nil {
			return err
		}

		reason := []byte(err.Error())
		if err = binary.Write(bw, binary.BigEndian, &reason); err != nil {
			return err
		}

		bw.Flush()
		return err
	}

	if err := binary.Write(*c.c, binary.BigEndian, uint32(0)); err != nil {
		return err
	}

	return nil
}
