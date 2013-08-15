package vnc

// Rectangle represents a rectangle of pixel data.
type Rectangle struct {
	X      uint16
	Y      uint16
	Width  uint16
	Height uint16
	Enc    Encoding
}
