package vnc

import (
	"encoding/binary"
	//	"fmt"
	"io"
)

// A ClientMessage implements a message sent from the client to the server.
type ClientMessage interface {
	// The type of the message that is sent down on the wire.
	Type() uint8

	// Read reads the contents of the message from the reader. At the point
	// this is called, the message type has already been read from the reader.
	// This should return a new ClientMessage that is the appropriate type.
	Read(*ServerConn, io.Reader) (ClientMessage, error)
}

// SetPixelFormat sets the format in which pixel values should be sent
// in FramebufferUpdate messages from the server.
//
// See RFC 6143 Section 7.5.1
type SetPixelFormatMessage struct {
	PixelFormat PixelFormat
}

func (*SetPixelFormatMessage) Type() uint8 {
	return 0
}

func (*SetPixelFormatMessage) Read(c *ServerConn, r io.Reader) (ClientMessage, error) {
	var result SetPixelFormatMessage

	// Read off the padding
	var padding [3]byte
	if _, err := io.ReadFull(r, padding[:]); err != nil {
		return nil, err
	}

	// Read the pixel format
	if err := readPixelFormat(c.c, &result.PixelFormat); err != nil {
		return nil, err
	}

	c.PixelFormat = result.PixelFormat
	c.config.PixelFormat = result.PixelFormat
	return &result, nil
}

// SetEncodings sets the encoding types in which the pixel data can
// be sent from the server. After calling this method, the encs slice
// given should not be modified.
//
// See RFC 6143 Section 7.5.2
type SetEncodingsMessage struct {
	Encs []Encoding
}

func (*SetEncodingsMessage) Type() uint8 {
	return 2
}

func (*SetEncodingsMessage) Read(c *ServerConn, r io.Reader) (ClientMessage, error) {
	var result SetEncodingsMessage

	// Read off the padding
	var padding [1]byte
	if _, err := io.ReadFull(r, padding[:]); err != nil {
		return nil, err
	}

	var encNumber uint16
	if err := binary.Read(r, binary.BigEndian, &encNumber); err != nil {
		return nil, err
	}
	// Build the map of encodings supported
	encMap := make(map[int32]Encoding)
	for _, enc := range c.Encs {
		encMap[enc.Type()] = enc
	}

	var encType int32
	for i := uint16(0); i < encNumber; i++ {
		if err := binary.Read(r, binary.BigEndian, &encType); err != nil {
			return nil, err
		}
	}

	rawEnc := new(RawEncoding)
	encMap[rawEnc.Type()] = rawEnc

	result.Encs = append(result.Encs, rawEnc)
	c.Encs = result.Encs

	return &result, nil
}

// FramebufferUpdateRequestMessage consists of a sequence of rectangles of
// pixel data that the client should put into its framebuffer.
type FramebufferUpdateRequestMessage struct {
	Incremental uint8
	X           uint16
	Y           uint16
	Width       uint16
	Height      uint16
}

// Requests a framebuffer update from the server. There may be an indefinite
// time between the request and the actual framebuffer update being
// received.
//
// See RFC 6143 Section 7.5.3
func (*FramebufferUpdateRequestMessage) Type() uint8 {
	return 3
}

func (*FramebufferUpdateRequestMessage) Read(c *ServerConn, r io.Reader) (ClientMessage, error) {
	var result FramebufferUpdateRequestMessage

	data := []interface{}{
		&result.Incremental,
		&result.X,
		&result.Y,
		&result.Width,
		&result.Height,
	}

	for _, val := range data {
		if err := binary.Read(r, binary.BigEndian, val); err != nil {
			return nil, err
		}
	}

	return &result, nil
}

// A KeyEvent message indicates a key press or release.  Down-flag is
// non-zero (true) if the key is now pressed, and zero (false) if it is
// now released.  The key itself is specified using the "keysym" values
// defined by the X Window System, even if the client or server is not
// running the X Window System
//
// See RFC 6143 Section 7.5.4
type KeyEventMessage struct {
	Down   uint8
	Keysym uint32
}

func (*KeyEventMessage) Type() uint8 {
	return 4
}

func (*KeyEventMessage) Read(c *ServerConn, r io.Reader) (ClientMessage, error) {
	var result KeyEventMessage

	if err := binary.Read(r, binary.BigEndian, &result.Down); err != nil {
		return nil, err
	}

	// Read off the padding
	var padding [2]byte
	if _, err := io.ReadFull(r, padding[:]); err != nil {
		return nil, err
	}

	if err := binary.Read(r, binary.BigEndian, &result.Keysym); err != nil {
		return nil, err
	}

	return &result, nil
}

// A PointerEvent message indicates either pointer movement or a pointer
// button press or release.
//
// See RFC 6143 Section 7.5.5
type PointerEventMessage struct {
	Mask uint8
	X    uint16
	Y    uint16
}

func (*PointerEventMessage) Type() uint8 {
	return 5
}

func (*PointerEventMessage) Read(c *ServerConn, r io.Reader) (ClientMessage, error) {
	var result PointerEventMessage

	data := []interface{}{
		&result.Mask,
		&result.X,
		&result.Y,
	}

	for _, val := range data {
		if err := binary.Read(r, binary.BigEndian, val); err != nil {
			return nil, err
		}
	}

	return &result, nil
}

// ClientCutTextMessage indicates the client has new text in the cut buffer.
//
// See RFC 6143 Section 7.5.6
type ClientCutTextMessage struct {
	Text string
}

func (*ClientCutTextMessage) Type() uint8 {
	return 6
}

func (*ClientCutTextMessage) Read(c *ServerConn, r io.Reader) (ClientMessage, error) {
	// Read off the padding
	var padding [1]byte
	if _, err := io.ReadFull(r, padding[:]); err != nil {
		return nil, err
	}

	var textLength uint32
	if err := binary.Read(r, binary.BigEndian, &textLength); err != nil {
		return nil, err
	}

	textBytes := make([]uint8, textLength)
	if err := binary.Read(r, binary.BigEndian, &textBytes); err != nil {
		return nil, err
	}

	return &ClientCutTextMessage{string(textBytes)}, nil
}
