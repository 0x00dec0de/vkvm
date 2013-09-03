package main

type PixelFormat struct {
	Bpp        uint8
	Depth      uint8
	BigEndian  uint8
	TrueColor  uint8
	RedMax     uint16
	GreenMax   uint16
	BlueMax    uint16
	RedShift   uint8
	GreenShift uint8
	BlueShift  uint8
	Padding    [3]uint8
}
