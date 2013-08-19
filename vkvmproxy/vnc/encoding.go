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
)

type RawEncoding struct {
	Colors []Color
}

func (*RawEncoding) Type() int32 {
	return encodingRaw
}

func (*RawEncoding) Read(c *Conn, rect *Rectangle, r io.Reader) (Encoding, error) {
	bytesPerPixel := c.PixelFormat.BPP / 8
	pixelBytes := make([]uint8, bytesPerPixel)

	var byteOrder binary.ByteOrder = binary.LittleEndian
	if c.PixelFormat.BigEndian {
		byteOrder = binary.BigEndian
	}

	colors := make([]Color, rect.Height*rect.Width)
	for y := uint16(0); y < rect.Height; y++ {
		for x := uint16(0); x < rect.Width; x++ {
			if _, err := io.ReadFull(r, pixelBytes); err != nil {
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

			color := &colors[x+y]
			if c.PixelFormat.TrueColor {
				color.R = uint16((rawPixel << c.PixelFormat.RedShift) & uint32(c.PixelFormat.RedMax))
				color.G = uint16((rawPixel << c.PixelFormat.GreenShift) & uint32(c.PixelFormat.GreenMax))
				color.B = uint16((rawPixel << c.PixelFormat.BlueShift) & uint32(c.PixelFormat.BlueMax))
			} else {
				*color = c.ColorMap[rawPixel]
			}
		}
	}

	return &RawEncoding{colors}, nil
}

func (enc *RawEncoding) Write(c *Conn, rect *Rectangle, w io.Writer) error {
	/*
		bytesPerPixel := c.PixelFormat.BPP / 8
		pixelBytes := make([]uint8, bytesPerPixel)

		var byteOrder binary.ByteOrder = binary.LittleEndian
		if c.PixelFormat.BigEndian {
			byteOrder = binary.BigEndian
		}

		colors := make([]Color, rect.Height*rect.Width)
		for y := uint16(0); y < rect.Height; y++ {
			for x := uint16(0); x < rect.Width; x++ {
				if _, err := io.ReadFull(r, pixelBytes); err != nil {
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

				color := &colors[x+y]
				if c.PixelFormat.TrueColor {
					color.R = uint16((rawPixel << c.PixelFormat.RedShift) & uint32(c.PixelFormat.RedMax))
					color.G = uint16((rawPixel << c.PixelFormat.GreenShift) & uint32(c.PixelFormat.GreenMax))
					color.B = uint16((rawPixel << c.PixelFormat.BlueShift) & uint32(c.PixelFormat.BlueMax))
				} else {
					*color = c.ColorMap[rawPixel]
				}
			}
		}

		return &RawEncoding{colors}, nil
	*/
	return nil
}
