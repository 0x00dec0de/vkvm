package vnc

type ButtonMask uint8

const (
	ButtonLeft ButtonMask = 1 << iota
	ButtonMiddle
	ButtonRight
	Button4
	Button5
	Button6
	Button7
	Button8
)
