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
	pixelBufferU32 []uint32
)

type RawEncoding struct {
	Colors []Color
}

func (*RawEncoding) Type() int32 {
	return encodingRaw
}

/*
func (*RawEncoding) Read(c *Conn, rect *Rectangle, r io.Reader) (Encoding, error) {
	m := &RawEncoding{}
	bytesPerPixel := c.PixelFormat.BPP / 8
	pixelBytes := make([]uint8, bytesPerPixel)

	var byteOrder binary.ByteOrder = binary.LittleEndian
	if c.PixelFormat.BigEndian {
		byteOrder = binary.BigEndian
	}

	colors := make([]Color, rect.Height*rect.Width)
	for y := uint16(0); y < rect.Height; y++ {
		for x := uint16(0); x < rect.Width; x++ {
			if err := binary.Read(r, byteOrder, &pixelBytes); err != nil {
				return nil, err
			}

			var rawPixel uint32
			if c.PixelFormat.BPP == 8 {
				rawPixel = uint32(pixelBytes[0])
			} else if c.PixelFormat.BPP == 16 {
				rawPixel = uint32(byteOrder.Uint16(pixelBytes))
			} else if c.PixelFormat.BPP == 32 {
				rawPixel = byteOrder.Uint32(pixelBytes)
			}
			index := x + y
			color := &colors[index]
			if c.PixelFormat.TrueColor {
				color.R = uint16((rawPixel >> c.PixelFormat.RedShift) & uint32(c.PixelFormat.RedMax))
				color.G = uint16((rawPixel >> c.PixelFormat.GreenShift) & uint32(c.PixelFormat.GreenMax))
				color.B = uint16((rawPixel >> c.PixelFormat.BlueShift) & uint32(c.PixelFormat.BlueMax))
			} else {
				*color = c.ColorMap[rawPixel]
			}
		}
	}
	m.Colors = colors

	return m, nil
}
*/
/*
func (enc *RawEncoding) Write(c *Conn, rect *Rectangle, w io.Writer) error {
	var byteOrder binary.ByteOrder = binary.LittleEndian
	if c.PixelFormat.BigEndian {
		byteOrder = binary.BigEndian
	}
	colors := enc.Colors
	for y := uint16(0); y < rect.Height; y++ {
		for x := uint16(0); x < rect.Width; x++ {
			var rawPixel uint32
			color := &colors[x+y]
			if c.PixelFormat.TrueColor {
				rawPixel = uint32(color.R<<c.PixelFormat.RedShift | color.G<<c.PixelFormat.GreenShift | color.B<<c.PixelFormat.BlueShift)
			} else {

			}
			var v interface{}
			switch c.PixelFormat.BPP {
			case 32:
				v = rawPixel
			case 16:
				v = uint16(rawPixel)
			case 8:
				v = uint8(rawPixel)
			default:
				return fmt.Errorf("TODO: BPP of %d", c.PixelFormat.BPP)
			}
			if err := binary.Write(w, byteOrder, v); err != nil {
				return err
			}
		}
	}

	return nil
}
*/
/*
var (
	pixbuf interface{}
)

func createBuf(tpe, size int) {
	switch tpe {
	case 8:
		pixbuf = make([]uint8, size)
	case 16:
		pixbuf = make([]uint16, size)
	case 32:
		pixbuf = make([]uint32, size)
	}
}

func checkBuf(tpe, size int) {
	switch pixbuf.(type) {
	case []uint8:
		if tpe != 8 || len(pixbuf.([]uint8)) != size {
			createBuf(tpe, size)
		}
	case []uint16:
		if tpe != 16 || len(pixbuf.([]uint16)) != size {
			createBuf(tpe, size)
		}
	case []uint32:
		if tpe != 32 || len(pixbuf.([]uint32)) != size {
			createBuf(tpe, size)
		}
	}
}
*/

func (*RawEncoding) Read(c *Conn, rect *Rectangle, r io.Reader) (Encoding, error) {
	m := &RawEncoding{}
	bufferSize := int(rect.Width * rect.Height)

	var byteOrder binary.ByteOrder = binary.LittleEndian
	if c.PixelFormat.BigEndian {
		byteOrder = binary.BigEndian
	}

	colors := make([]Color, int(rect.Height*rect.Width))

	switch {

	case c.PixelFormat.TrueColor == false:

		// Todo

	case c.PixelFormat.TrueColor && c.PixelFormat.BPP == 8:

		// Todo

	case c.PixelFormat.TrueColor && c.PixelFormat.BPP == 16:

		// Todo

	case c.PixelFormat.TrueColor && c.PixelFormat.BPP == 32:

		if len(pixelBufferU32) != bufferSize {
			pixelBufferU32 = make([]uint32, bufferSize)
		}

		if err := binary.Read(r, byteOrder, &pixelBufferU32); err != nil {
			return nil, err
		}

		for y := uint16(0); y < rect.Height; y++ {
			for x := uint16(0); x < rect.Width; x++ {
				index := x + y*rect.Width
				rawPixel := pixelBufferU32[index]
				color := &colors[index]
				color.R = uint16(rawPixel>>c.PixelFormat.RedShift) & c.PixelFormat.RedMax
				color.G = uint16(rawPixel>>c.PixelFormat.GreenShift) & c.PixelFormat.GreenMax
				color.B = uint16(rawPixel>>c.PixelFormat.BlueShift) & c.PixelFormat.BlueMax
			}
		}
	}

	m.Colors = colors
	return m, nil
}

func (enc *RawEncoding) Write(c *Conn, rect *Rectangle, w io.Writer) error {
	var byteOrder binary.ByteOrder = binary.LittleEndian
	if c.PixelFormat.BigEndian {
		byteOrder = binary.BigEndian
	}
	colors := enc.Colors

	bufferSize := int(rect.Width * rect.Height)
	if len(pixelBufferU32) != bufferSize {
		pixelBufferU32 = make([]uint32, bufferSize)
	}

	for y := uint16(0); y < rect.Height; y++ {
		for x := uint16(0); x < rect.Width; x++ {
			index := x + y*rect.Width
			color := &colors[index]
			switch {

			case c.PixelFormat.TrueColor == false:

				// Todo

			case c.PixelFormat.TrueColor && c.PixelFormat.BPP == 8:

				// Todo

			case c.PixelFormat.TrueColor && c.PixelFormat.BPP == 16:

				// Todo

			case c.PixelFormat.TrueColor && c.PixelFormat.BPP == 32:
				pixelBufferU32[index] = uint32(color.R)<<c.PixelFormat.RedShift | uint32(color.G)<<c.PixelFormat.GreenShift | uint32(color.B)<<c.PixelFormat.BlueShift
			}
		}
	}
	if err := binary.Write(w, byteOrder, pixelBufferU32); err != nil {
		return err
	}
	return nil
}
