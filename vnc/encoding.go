package vnc

import (
	"encoding/binary"
	"io"
)

type Encoding interface {
	Type() int32

	Read(*Conn, *Rectangle, io.Reader) (Encoding, error)
	Write(*Conn, *Rectangle, io.Writer) error
}

const (
	encodingRaw = iota
	encodingCopyRect
	encodingDesktopSize = -223
)

type DesktopSizeEncoding struct {
	Width  uint16
	Height uint16
}

func (*DesktopSizeEncoding) Type() int32 {
	return encodingDesktopSize
}

func (*DesktopSizeEncoding) Read(c *Conn, rect *Rectangle, r io.Reader) (Encoding, error) {
	return nil, nil
}

func (*DesktopSizeEncoding) Write(c *Conn, rect *Rectangle, w io.Writer) error {

	return nil
}

var (
	pixelBufferWU32 []uint32
	pixelBufferRU32 []uint32
)

type RawEncoding struct {
	Colors []Color
}

func (*RawEncoding) Type() int32 {
	return encodingRaw
}

func (*RawEncoding) Read(c *Conn, rect *Rectangle, r io.Reader) (Encoding, error) {
	m := &RawEncoding{}
	var byteOrder binary.ByteOrder = binary.LittleEndian

	bufferSize := int(rect.Width * rect.Height)
	colors := make([]Color, bufferSize)

	if c.PixelFormat.BigEndian {
		byteOrder = binary.BigEndian
	}

	switch {
	case c.PixelFormat.TrueColor == false:
		// Todo
	case c.PixelFormat.TrueColor && c.PixelFormat.BPP == 8:
		// Todo
	case c.PixelFormat.TrueColor && c.PixelFormat.BPP == 16:
		// Todo
	case c.PixelFormat.TrueColor && c.PixelFormat.BPP == 32:
		if len(pixelBufferRU32) != bufferSize {
			pixelBufferRU32 = make([]uint32, bufferSize)
		}

		if err := binary.Read(r, byteOrder, &pixelBufferRU32); err != nil {
			return nil, err
		}

		for index, rawPixel := range pixelBufferRU32 {
			color := &colors[index]

			color.R = uint16(rawPixel>>c.PixelFormat.RedShift) & c.PixelFormat.RedMax
			color.G = uint16(rawPixel>>c.PixelFormat.GreenShift) & c.PixelFormat.GreenMax
			color.B = uint16(rawPixel>>c.PixelFormat.BlueShift) & c.PixelFormat.BlueMax
		}
	}

	m.Colors = colors
	return m, nil
}

func (enc *RawEncoding) Write(c *Conn, rect *Rectangle, w io.Writer) error {
	var byteOrder binary.ByteOrder = binary.LittleEndian

	bufferSize := int(rect.Width * rect.Height)
	colors := enc.Colors

	if c.PixelFormat.BigEndian {
		byteOrder = binary.BigEndian
	}

	switch {
	case c.PixelFormat.TrueColor == false:
		// Todo
	case c.PixelFormat.TrueColor && c.PixelFormat.BPP == 8:
		// Todo
	case c.PixelFormat.TrueColor && c.PixelFormat.BPP == 16:
		// Todo
	case c.PixelFormat.TrueColor && c.PixelFormat.BPP == 32:
		if len(pixelBufferWU32) != bufferSize {
			pixelBufferWU32 = make([]uint32, bufferSize)
		}
	}

	for index, _ := range pixelBufferWU32 {
		color := &colors[index]
		switch {
		case c.PixelFormat.TrueColor == false:
			// Todo
		case c.PixelFormat.TrueColor && c.PixelFormat.BPP == 8:
			// Todo
		case c.PixelFormat.TrueColor && c.PixelFormat.BPP == 16:
			// Todo
		case c.PixelFormat.TrueColor && c.PixelFormat.BPP == 32:
			pixelBufferWU32[index] = uint32(color.R)<<c.PixelFormat.RedShift | uint32(color.G)<<c.PixelFormat.GreenShift | uint32(color.B)<<c.PixelFormat.BlueShift
		}
	}

	if err := binary.Write(w, byteOrder, pixelBufferWU32); err != nil {
		return err
	}
	return nil
}
