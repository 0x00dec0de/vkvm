package vnc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// A ServerMessage implements a message sent from the server to the client.
type ServerMessage interface {
	// The type of the message that is sent down on the wire.
	Type() uint8

	// Read reads the contents of the message from the reader. At the point
	// this is called, the message type has already been read from the reader.
	// This should return a new ServerMessage that is the appropriate type.
	Read(*ClientConn, io.Reader) (ServerMessage, error)
	Write(*ClientConn, io.Writer) ([]byte, error)
}

// FramebufferUpdateMessage consists of a sequence of rectangles of
// pixel data that the client should put into its framebuffer.
type FramebufferUpdateMessage struct {
	Rectangles []Rectangle
}

func (*FramebufferUpdateMessage) Type() uint8 {
	return 0
}

func (m *FramebufferUpdateMessage) Write(c *ClientConn, w io.Writer) (result []byte, err error) {
	var padding [1]byte

	buffer := new(bytes.Buffer)
	if err := binary.Write(buffer, binary.BigEndian, padding); err != nil {
		return nil, err
	}

	numRects := cap(m.Rectangles)
	if err := binary.Write(buffer, binary.BigEndian, numRects); err != nil {
		return nil, err
	}

	// Build the map of encodings supported
	encMap := make(map[int32]Encoding)
	for _, enc := range c.Encs {
		encMap[enc.Type()] = enc
	}

	// We must always support the raw encoding
	rawEnc := new(RawEncoding)
	encMap[rawEnc.Type()] = rawEnc

	rects := make([]Rectangle, numRects)
	for i := uint16(0); i < numRects; i++ {
		var encodingType int32

		rect := &rects[i]
		data := []interface{}{
			&rect.X,
			&rect.Y,
			&rect.Width,
			&rect.Height,
			&encodingType,
		}

		for _, val := range data {
			if err := binary.Write(buffer, binary.BigEndian, val); err != nil {
				return nil, err
			}

			enc, ok := encMap[encodingType]
			if !ok {
				return nil, fmt.Errorf("unsupported encoding type: %d", encodingType)
			}

			var err error
			rect.Enc, err = enc.Write(c, rect, w)
			if err != nil {
				return nil, err
			}

		}

	}
	return nil, nil
}

func (*FramebufferUpdateMessage) Read(c *ClientConn, r io.Reader) (ServerMessage, error) {
	// Read off the padding
	var padding [1]byte
	if _, err := io.ReadFull(r, padding[:]); err != nil {
		return nil, err
	}

	var numRects uint16
	if err := binary.Read(r, binary.BigEndian, &numRects); err != nil {
		return nil, err
	}

	// Build the map of encodings supported
	encMap := make(map[int32]Encoding)
	for _, enc := range c.Encs {
		encMap[enc.Type()] = enc
	}

	// We must always support the raw encoding
	rawEnc := new(RawEncoding)
	encMap[rawEnc.Type()] = rawEnc

	rects := make([]Rectangle, numRects)
	for i := uint16(0); i < numRects; i++ {
		var encodingType int32

		rect := &rects[i]
		data := []interface{}{
			&rect.X,
			&rect.Y,
			&rect.Width,
			&rect.Height,
			&encodingType,
		}

		for _, val := range data {
			if err := binary.Read(r, binary.BigEndian, val); err != nil {
				return nil, err
			}
		}

		enc, ok := encMap[encodingType]
		if !ok {
			return nil, fmt.Errorf("unsupported encoding type: %d", encodingType)
		}

		var err error
		rect.Enc, err = enc.Read(c, rect, r)
		if err != nil {
			return nil, err
		}
	}

	return &FramebufferUpdateMessage{rects}, nil
}

// SetColorMapEntriesMessage is sent by the server to set values into
// the color map. This message will automatically update the color map
// for the associated connection, but contains the color change data
// if the consumer wants to read it.
//
// See RFC 6143 Section 7.6.2
type SetColorMapEntriesMessage struct {
	FirstColor uint16
	Colors     []Color
}

func (*SetColorMapEntriesMessage) Type() uint8 {
	return 1
}

func (m *SetColorMapEntriesMessage) Write(c *ClientConn, w io.Writer) ([]byte, error) {
	return nil, nil
}

func (*SetColorMapEntriesMessage) Read(c *ClientConn, r io.Reader) (ServerMessage, error) {
	// Read off the padding
	var padding [1]byte
	if _, err := io.ReadFull(r, padding[:]); err != nil {
		return nil, err
	}

	var result SetColorMapEntriesMessage
	if err := binary.Read(r, binary.BigEndian, &result.FirstColor); err != nil {
		return nil, err
	}

	var numColors uint16
	if err := binary.Read(r, binary.BigEndian, &numColors); err != nil {
		return nil, err
	}

	result.Colors = make([]Color, numColors)
	for i := uint16(0); i < numColors; i++ {

		color := &result.Colors[i]
		data := []interface{}{
			&color.R,
			&color.G,
			&color.B,
		}

		for _, val := range data {
			if err := binary.Read(r, binary.BigEndian, val); err != nil {
				return nil, err
			}
		}

		// Update the connection's color map
		c.ColorMap[result.FirstColor+i] = *color
	}

	return &result, nil
}

// Bell signals that an audible bell should be made on the client.
//
// See RFC 6143 Section 7.6.3
type BellMessage byte

func (*BellMessage) Type() uint8 {
	return 2
}

func (m *BellMessage) Write(c *ClientConn, w io.Writer) ([]byte, error) {
	return nil, nil
}

func (*BellMessage) Read(*ClientConn, io.Reader) (ServerMessage, error) {
	return new(BellMessage), nil
}

// ServerCutTextMessage indicates the server has new text in the cut buffer.
//
// See RFC 6143 Section 7.6.4
type ServerCutTextMessage struct {
	Text string
}

func (*ServerCutTextMessage) Type() uint8 {
	return 3
}

func (m *ServerCutTextMessage) Write(c *ClientConn, w io.Writer) ([]byte, error) {
	return nil, nil
}

func (*ServerCutTextMessage) Read(c *ClientConn, r io.Reader) (ServerMessage, error) {
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

	return &ServerCutTextMessage{string(textBytes)}, nil
}
