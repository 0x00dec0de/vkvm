package vnc

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
)

type Message interface {
	Type() uint8
	Read(*Conn, io.Reader) (Message, error)
	Write(*Conn, io.Writer) error
}

type SetPixelFormatMsg struct {
	PixelFormat *PixelFormat
}

func (msg *SetPixelFormatMsg) Type() uint8 {
	return 0
}

func (msg *SetPixelFormatMsg) Read(c *Conn, r io.Reader) (Message, error) {
	var err error
	m := &SetPixelFormatMsg{}
	// Read off the padding
	var padding [3]uint8
	if err = binary.Read(r, binary.BigEndian, &padding); err != nil {
		return nil, err
	}
	var format *PixelFormat
	// Read the pixel format
	if format, err = readPixelFormat(r); err != nil {
		return nil, err
	}
	m.PixelFormat = format
	c.PixelFormat = m.PixelFormat
	return m, nil
}

func (msg *SetPixelFormatMsg) Write(c *Conn, w io.Writer) error {
	pixelFormat, err := writePixelFormat(msg.PixelFormat)
	if err != nil {
		return err
	}
	bw := bufio.NewWriter(w)

	data := []interface{}{
		uint8(0),
		uint8(0),
		uint8(0),
		uint8(0),
		pixelFormat,
	}
	for _, val := range data {
		if err := binary.Write(bw, binary.BigEndian, val); err != nil {
			return err
		}
	}
	bw.Flush()
	return nil
}

type SetEncodingsMsg struct {
	Encs []Encoding
}

func (msg *SetEncodingsMsg) Type() uint8 {
	return 2
}

func (msg *SetEncodingsMsg) Read(c *Conn, r io.Reader) (Message, error) {
	m := &SetEncodingsMsg{}

	var padding [1]byte
	if err := binary.Read(r, binary.BigEndian, &padding); err != nil {
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

	m.Encs = append(m.Encs, rawEnc)
	c.Encs = m.Encs

	return m, nil
}

func (msg *SetEncodingsMsg) Write(c *Conn, w io.Writer) error {
	bw := bufio.NewWriter(w)

	encNumber := uint16(len(msg.Encs))
	data := []interface{}{
		uint8(2),
		uint8(0),
		encNumber,
	}

	for _, val := range data {
		if err := binary.Write(bw, binary.BigEndian, val); err != nil {
			return err
		}
	}

	for i := uint16(0); i < encNumber; i++ {
		if err := binary.Write(bw, binary.BigEndian, msg.Encs[i].Type()); err != nil {
			return err
		}
	}

	bw.Flush()
	return nil
}

type FramebufferUpdateRequestMsg struct {
	Incremental uint8
	X           uint16
	Y           uint16
	Width       uint16
	Height      uint16
}

func (msg *FramebufferUpdateRequestMsg) Type() uint8 {
	return 3
}

func (msg *FramebufferUpdateRequestMsg) Read(c *Conn, r io.Reader) (Message, error) {
	m := &FramebufferUpdateRequestMsg{}

	data := []interface{}{
		&m.Incremental,
		&m.X,
		&m.Y,
		&m.Width,
		&m.Height,
	}

	for _, val := range data {
		if err := binary.Read(r, binary.BigEndian, val); err != nil {
			return nil, err
		}
	}

	return m, nil
}

func (msg *FramebufferUpdateRequestMsg) Write(c *Conn, w io.Writer) error {
	bw := bufio.NewWriter(w)
	data := []interface{}{
		uint8(3),
		msg.Incremental,
		msg.X,
		msg.Y,
		msg.Width,
		msg.Height,
	}

	for _, val := range data {
		if err := binary.Write(bw, binary.BigEndian, val); err != nil {
			return err
		}
	}

	bw.Flush()
	return nil
}

type KeyEventMsg struct {
	Down   uint8
	Keysym uint32
}

func (msg *KeyEventMsg) Type() uint8 {
	return 4
}

func (msg *KeyEventMsg) Read(c *Conn, r io.Reader) (Message, error) {
	m := &KeyEventMsg{}

	if err := binary.Read(r, binary.BigEndian, &m.Down); err != nil {
		return nil, err
	}

	// Read off the padding
	var padding [2]byte
	if err := binary.Read(r, binary.BigEndian, &padding); err != nil {
		return nil, err
	}

	if err := binary.Read(r, binary.BigEndian, &m.Keysym); err != nil {
		return nil, err
	}

	return m, nil
}

func (msg *KeyEventMsg) Write(c *Conn, w io.Writer) error {
	bw := bufio.NewWriter(w)

	data := []interface{}{
		uint8(4),
		msg.Down,
		uint8(0),
		uint8(0),
		msg.Keysym,
	}

	for _, val := range data {
		if err := binary.Write(bw, binary.BigEndian, val); err != nil {
			return err
		}
	}

	bw.Flush()
	return nil
}

type PointerEventMsg struct {
	Mask uint8
	X    uint16
	Y    uint16
}

func (msg *PointerEventMsg) Type() uint8 {
	return 5
}

func (msg *PointerEventMsg) Read(c *Conn, r io.Reader) (Message, error) {
	m := &PointerEventMsg{}

	data := []interface{}{
		&m.Mask,
		&m.X,
		&m.Y,
	}

	for _, val := range data {
		if err := binary.Read(r, binary.BigEndian, val); err != nil {
			return nil, err
		}
	}

	return m, nil
}

func (msg *PointerEventMsg) Write(c *Conn, w io.Writer) error {
	bw := bufio.NewWriter(w)

	data := []interface{}{
		uint8(5),
		msg.Mask,
		msg.X,
		msg.Y,
	}

	for _, val := range data {
		if err := binary.Write(bw, binary.BigEndian, val); err != nil {
			return err
		}
	}

	bw.Flush()
	return nil
}

type ClientCutTextMsg struct {
	Text string
}

func (msg *ClientCutTextMsg) Type() uint8 {
	return 6
}

func (msg *ClientCutTextMsg) Read(c *Conn, r io.Reader) (Message, error) {
	m := &ClientCutTextMsg{}
	// Read off the padding
	var padding [3]byte
	if err := binary.Read(r, binary.BigEndian, &padding); err != nil {
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

	m.Text = string(textBytes)
	return m, nil
}

func (msg *ClientCutTextMsg) Write(c *Conn, w io.Writer) error {
	bw := bufio.NewWriter(w)

	textBytes := []byte(msg.Text)
	textLength := uint32(len(textBytes))

	data := []interface{}{
		uint8(6),
		uint8(0),
		uint8(0),
		uint8(0),
		textLength,
	}

	for _, val := range data {
		if err := binary.Write(bw, binary.BigEndian, val); err != nil {
			return err
		}
	}

	if err := binary.Write(bw, binary.BigEndian, textBytes); err != nil {
		return err
	}

	bw.Flush()
	return nil
}

type FramebufferUpdateMsg struct {
	Rectangles []Rectangle
}

func (msg *FramebufferUpdateMsg) Type() uint8 {
	return 0
}

func (msg *FramebufferUpdateMsg) Read(c *Conn, r io.Reader) (Message, error) {
	m := &FramebufferUpdateMsg{}
	// Read off the padding
	var padding [1]byte
	if err := binary.Read(r, binary.BigEndian, &padding); err != nil {
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
			return nil, errors.New("unsupported encoding type") //, encodingType)
		}

		var err error
		rect.Enc, err = enc.Read(c, rect, r)
		if err != nil {
			return nil, err
		}
	}

	m.Rectangles = rects
	return m, nil
}

func (msg *FramebufferUpdateMsg) Write(c *Conn, w io.Writer) error {
	bw := bufio.NewWriter(w)
	numRects := uint16(len(msg.Rectangles))

	data := []interface{}{
		uint8(0),
		uint8(0),
		numRects,
	}

	for _, val := range data {
		if err := binary.Write(bw, binary.BigEndian, val); err != nil {
			return err
		}
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
			if err := binary.Write(bw, binary.BigEndian, val); err != nil {
				return err
			}

			enc, ok := encMap[encodingType]
			if !ok {
				return errors.New("unsupported encoding type") //, encodingType)
			}

			var err error
			if err = enc.Write(c, rect, w); err != nil {
				return err
			}

		}

	}
	return nil
}

type SetColorMapEntriesMsg struct {
	FirstColor uint16
	Colors     []Color
}

func (msg *SetColorMapEntriesMsg) Type() uint8 {
	return 1
}

func (msg *SetColorMapEntriesMsg) Read(c *Conn, r io.Reader) (Message, error) {
	var padding [1]byte
	m := &SetColorMapEntriesMsg{}
	if err := binary.Read(r, binary.BigEndian, &padding); err != nil {
		return nil, err
	}

	if err := binary.Read(r, binary.BigEndian, &m.FirstColor); err != nil {
		return nil, err
	}

	var numColors uint16
	if err := binary.Read(r, binary.BigEndian, &numColors); err != nil {
		return nil, err
	}

	m.Colors = make([]Color, numColors)
	for i := uint16(0); i < numColors; i++ {
		color := &m.Colors[i]
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

		c.ColorMap[m.FirstColor+i] = *color
	}

	return m, nil
}

func (msg *SetColorMapEntriesMsg) Write(c *Conn, w io.Writer) error {
	bw := bufio.NewWriter(w)
	numColors := uint16(len(msg.Colors))

	data := []interface{}{
		uint8(1),
		uint8(0),
		msg.FirstColor,
		numColors,
	}

	for _, val := range data {
		if err := binary.Write(bw, binary.BigEndian, val); err != nil {
			return err
		}
	}

	for i := uint16(0); i < numColors; i++ {
		color := &msg.Colors[i]
		data := []interface{}{
			color.R,
			color.G,
			color.B,
		}

		for _, val := range data {
			if err := binary.Write(bw, binary.BigEndian, val); err != nil {
				return err
			}
		}
	}

	bw.Flush()
	return nil
}

type BellMsg byte

func (*BellMsg) Type() uint8 {
	return 2
}

func (msg *BellMsg) Write(c *Conn, w io.Writer) error {
	return binary.Write(w, binary.BigEndian, uint8(2))
}

func (msg *BellMsg) Read(c *Conn, r io.Reader) (Message, error) {
	return new(BellMsg), nil
}

type ServerCutTextMsg struct {
	Text string
}

func (msg *ServerCutTextMsg) Type() uint8 {
	return 3
}

func (msg *ServerCutTextMsg) Write(c *Conn, w io.Writer) error {
	bw := bufio.NewWriter(w)

	textBytes := []byte(msg.Text)
	textLength := uint32(len(textBytes))
	data := []interface{}{
		uint8(3),
		uint8(0),
		uint8(0),
		uint8(0),
		textLength,
	}

	for _, val := range data {
		if err := binary.Write(bw, binary.BigEndian, val); err != nil {
			return err
		}
	}

	if err := binary.Write(bw, binary.BigEndian, textBytes); err != nil {
		return err
	}

	bw.Flush()
	return nil
}

func (msg *ServerCutTextMsg) Read(c *Conn, r io.Reader) (Message, error) {
	m := &ServerCutTextMsg{}
	var padding [1]byte
	if err := binary.Read(r, binary.BigEndian, &padding); err != nil {
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

	m.Text = string(textBytes)
	return m, nil
}
