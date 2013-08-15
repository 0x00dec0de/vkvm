// Package vnc implements a VNC client.
//
// References:
//   [PROTOCOL]: http://tools.ietf.org/html/rfc6143
package vnc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"unicode"
)

type Server struct {
	conns  chan *ServerConn
	Conns  <-chan *ServerConn
	config *ServerConfig
}

type ServerConn struct {
	// Server config
	s *Server

	// net.Conn
	c net.Conn

	// If the pixel format uses a color map, then this is the color
	// map that is used. This should not be modified directly, since
	// the data comes from the server.
	ColorMap [256]Color

	// Encodings supported by the client. This should not be modified
	// directly. Instead, SetEncodings should be used.
	Encs []Encoding

	// Name associated with the desktop, sent from the server.
	DesktopName string

	// The pixel format associated with the connection. This shouldn't
	// be modified. If you wish to set a new pixel format, use the
	// SetPixelFormat method.
	PixelFormat PixelFormat

	// The channels that all messages received from the client and sended
	// to server. If the channel blocks, then the goroutine reading data
	// from the VNC server may block indefinitely. It is up to the user
	// of the library to ensure that this channel is properly read.
	// If this is not set, then all messages will be discarded.
	MessageInp chan ClientMessage
	MessageOut chan ServerMessage

	// Exclusive determines whether the connection is shared with other
	// clients. If true, then all other clients connected will be
	// disconnected when a connection is established to the VNC server.
	Exclusive bool
}

// A ServerConfig structure is used to configure a ServerConn. After
// one has been passed to initialize a connection, it must not be modified.
type ServerConfig struct {
	// A slice of AuthType methods. Only the first instance that is
	// suitable by the server and client will be used to authenticate.
	AuthTypes []AuthType

	// A slice of supported messages that can be read from the server.
	// This only needs to contain NEW server messages, and doesn't
	// need to explicitly contain the RFC-required messages.
	ClientMessages []ClientMessage
	ServerMessages []ServerMessage

	DesktopName string

	ServerInit func(*ServerConn) error

	Width  int32
	Height int32

	MaxConn    int32
	MaxConnMsg int32
}

func NewServer(cfg *ServerConfig) *Server {
	conns := make(chan *ServerConn, cfg.MaxConn)
	return &Server{config: cfg, Conns: conns, conns: conns}
}

func (s *Server) Serve(ln net.Listener) error {
	for {
		c, err := ln.Accept()
		if err != nil {
			return err
		}
		conn := s.newConn(c)
		if err := conn.versionHandshake(); err != nil {
			conn.Close()
			return err
		}
		if err := conn.securityHandshake(); err != nil {
			conn.Close()
			return err
		}
		if err := conn.serverInit(); err != nil {
			conn.Close()
			return err
		}
		select {
		case s.conns <- conn:
		default:
			// client is behind; doesn't get this updated.
		}
		go conn.serve()
	}
	panic("unreachable")
}

func (s *Server) newConn(c net.Conn) *ServerConn {
	conn := &ServerConn{
		s:          s,
		c:          c,
		MessageInp: make(chan ClientMessage, s.config.MaxConnMsg),
		MessageOut: make(chan ServerMessage, s.config.MaxConnMsg),
	}
	return conn
}

func (c *ServerConn) Close() error {
	return c.c.Close()
}

// CutText tells the client that the server has new text in its cut buffer.
// The text string MUST only contain Latin-1 characters. This encoding
// is compatible with Go's native string format, but can only use up to
// unicode.MaxLatin values.
//
// See RFC 6143 Section 7.5.6
func (c *ServerConn) CutText(text string) error {
	var buf bytes.Buffer

	// This is the fixed size data we'll send
	fixedData := []interface{}{
		uint8(3),
		uint8(0),
		uint8(0),
		uint8(0),
		uint32(len(text)),
	}

	for _, val := range fixedData {
		if err := binary.Write(&buf, binary.BigEndian, val); err != nil {
			return err
		}
	}

	for _, char := range text {
		if char > unicode.MaxLatin1 {
			return fmt.Errorf("Character '%s' is not valid Latin-1", char)
		}

		if err := binary.Write(&buf, binary.BigEndian, uint8(char)); err != nil {
			return err
		}
	}

	dataLength := 8 + len(text)
	if _, err := c.c.Write(buf.Bytes()[0:dataLength]); err != nil {
		return err
	}

	return nil
}

func (c *ServerConn) Bell() error {
	if err := binary.Write(c.c, binary.BigEndian, uint8(2)); err != nil {
		return err
	}
	return nil
}

func (c *ServerConn) FramebufferUpdate(rectangles []Rectangle) error {
	var buf bytes.Buffer

	// This is the fixed size data we'll send
	fixedData := []interface{}{
		uint8(0),
		uint8(0),
		uint16(cap(rectangles)),
	}

	for _, val := range fixedData {
		if err := binary.Write(&buf, binary.BigEndian, val); err != nil {
			return err
		}
	}

	for _, rectangle := range rectangles {
		if err := binary.Write(&buf, binary.BigEndian, rectangle); err != nil {
			return err
		}
	}

	if _, err := c.c.Write(buf.Bytes()); err != nil {
		return err
	}

	return nil
}

func (c *ServerConn) SetColorMapEntries(firstColor uint16, colors []Color) error {
	var buf bytes.Buffer

	// This is the fixed size data we'll send
	fixedData := []interface{}{
		uint8(1),
		uint8(0),
		firstColor,
		uint16(cap(colors)),
	}

	for _, val := range fixedData {
		if err := binary.Write(&buf, binary.BigEndian, val); err != nil {
			return err
		}
	}

	for _, color := range colors {
		if err := binary.Write(&buf, binary.BigEndian, color); err != nil {
			return err
		}

		/*
			if err := binary.Write(&buf, binary.BigEndian, color.R); err != nil {
				return err
			}
			if err := binary.Write(&buf, binary.BigEndian, color.G); err != nil {
				return err
			}
			if err := binary.Write(&buf, binary.BigEndian, color.B); err != nil {
				return err
			}
		*/
	}

	if _, err := c.c.Write(buf.Bytes()); err != nil {
		return err
	}

	return nil
}

func (c *ServerConn) versionHandshake() error {
	var protocolVersion [12]byte

	// Respond with the version we will support
	if _, err := c.c.Write([]byte("RFB 003.008\n")); err != nil {
		return err
	}

	// 7.1.1, read the ProtocolVersion message sent by the server.
	if _, err := io.ReadFull(c.c, protocolVersion[:]); err != nil {
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
	return nil
}

func (c *ServerConn) securityHandshake() error {
	serverSecurityTypes := c.s.config.AuthTypes
	if serverSecurityTypes == nil {
		serverSecurityTypes = []AuthType{new(AuthTypeNone)}
	}

	var sectypes []uint8
	sectypes = []uint8{uint8(len(serverSecurityTypes))}
	for _, curAuth := range serverSecurityTypes {
		sectypes = append(sectypes, curAuth.Type())
	}
	if err := binary.Write(c.c, binary.BigEndian, sectypes); err != nil {
		return err
	}

	var securityType uint8
	if err := binary.Read(c.c, binary.BigEndian, &securityType); err != nil {
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
		return fmt.Errorf("no suitable auth schemes found. server supported: %#v", serverSecurityTypes)
	}

	if err := authType.Handler(c, c.c); err != nil {
		if err = binary.Write(c.c, binary.BigEndian, uint32(1)); err != nil {
			return err
		}
		c.writeErrorReason(err)
		return err
	}

	if err := binary.Write(c.c, binary.BigEndian, uint32(0)); err != nil {
		return err
	}

	return nil
}

func (c *ServerConn) serverInit() error {
	if c.s.config.ServerInit == nil {
		return serverInitDefault(c)
	}

	return c.s.config.ServerInit(c)
}

// mainLoop reads messages sent from the server and routes them to the
// proper channels for users of the client to read.
func (c *ServerConn) serve() {
	var err error
	defer c.Close()

	// Build the map of available client messages
	typeMap := make(map[uint8]ClientMessage)
	defaultMessages := []ClientMessage{
		new(SetPixelFormatMessage),
		new(SetEncodingsMessage),
		new(FramebufferUpdateRequestMessage),
		new(KeyEventMessage),
		new(PointerEventMessage),
		new(ClientCutTextMessage),
	}

	for _, msg := range defaultMessages {
		typeMap[msg.Type()] = msg
	}

	if c.s.config.ClientMessages != nil {
		for _, msg := range c.s.config.ClientMessages {
			typeMap[msg.Type()] = msg
		}
	}

	go func() {
		for {
			var messageType uint8
			if err = binary.Read(c.c, binary.BigEndian, &messageType); err != nil {
				break
			}
			msg, ok := typeMap[messageType]
			if !ok {
				// Unsupported message type! Bad!
				break
			}
			parsedMsg, err := msg.Read(c)
			if err != nil {
				break
			}

			if c.MessageInp == nil {
				continue
			}

			c.MessageInp <- parsedMsg
		}
	}()

	go func() {
		for {
			select {
			case msg := <-c.MessageOut:
				binaryMsg, err := msg.Write(c, c.c)
				if err != nil {
					break
				}
				if c.MessageOut == nil {
					continue
				}
				err = binary.Write(c.c, binary.BigEndian, binaryMsg)
				if err != nil {
					break
				}
			}
		}
	}()
	select {}
}

func (c *ServerConn) writeErrorReason(err error) {
	reasonLen := uint32(len(err.Error()))
	if err := binary.Write(c.c, binary.BigEndian, reasonLen); err != nil {
		return
	}

	reason := []byte(err.Error())
	if err := binary.Write(c.c, binary.BigEndian, &reason); err != nil {
		return
	}

	return
}
