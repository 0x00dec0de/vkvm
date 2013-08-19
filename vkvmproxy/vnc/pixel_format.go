package vnc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type PixelFormat struct {
	BPP        uint8
	Depth      uint8
	BigEndian  bool
	TrueColor  bool
	RedMax     uint16
	GreenMax   uint16
	BlueMax    uint16
	RedShift   uint8
	GreenShift uint8
	BlueShift  uint8
}

func readPixelFormat(r io.Reader) (*PixelFormat, error) {
	var err error
	var result PixelFormat
	var rawPixelFormat [16]byte
	if _, err = io.ReadFull(r, rawPixelFormat[:]); err != nil {
		return nil, err
	}
	fmt.Printf("%+v\n", rawPixelFormat)
	var pfBoolByte uint8
	brPF := bytes.NewReader(rawPixelFormat[:])
	if err = binary.Read(brPF, binary.BigEndian, &result.BPP); err != nil {
		return nil, err
	}

	if err = binary.Read(brPF, binary.BigEndian, &result.Depth); err != nil {
		return nil, err
	}

	if err = binary.Read(brPF, binary.BigEndian, &pfBoolByte); err != nil {
		return nil, err
	}

	if pfBoolByte != 0 {
		result.BigEndian = true
	}

	if err = binary.Read(brPF, binary.BigEndian, &pfBoolByte); err != nil {
		return nil, err
	}

	if pfBoolByte != 0 {
		result.TrueColor = true

		if err := binary.Read(brPF, binary.BigEndian, &result.RedMax); err != nil {
			return nil, err
		}

		if err := binary.Read(brPF, binary.BigEndian, &result.GreenMax); err != nil {
			return nil, err
		}

		if err := binary.Read(brPF, binary.BigEndian, &result.BlueMax); err != nil {
			return nil, err
		}

		if err := binary.Read(brPF, binary.BigEndian, &result.RedShift); err != nil {
			return nil, err
		}

		if err := binary.Read(brPF, binary.BigEndian, &result.GreenShift); err != nil {
			return nil, err
		}

		if err := binary.Read(brPF, binary.BigEndian, &result.BlueShift); err != nil {
			return nil, err
		}
	}

	return &result, nil
}

func writePixelFormat(format *PixelFormat) ([]byte, error) {
	var buf bytes.Buffer

	// Byte 1
	if err := binary.Write(&buf, binary.BigEndian, format.BPP); err != nil {
		return nil, err
	}

	// Byte 2
	if err := binary.Write(&buf, binary.BigEndian, format.Depth); err != nil {
		return nil, err
	}

	var boolByte byte
	if format.BigEndian {
		boolByte = 1
	} else {
		boolByte = 0
	}

	// Byte 3 (BigEndian)
	if err := binary.Write(&buf, binary.BigEndian, boolByte); err != nil {
		return nil, err
	}

	if format.TrueColor {
		boolByte = 1
	} else {
		boolByte = 0
	}

	// Byte 4 (TrueColor)
	if err := binary.Write(&buf, binary.BigEndian, boolByte); err != nil {
		return nil, err
	}

	// If we have true color enabled then we have to fill in the rest of the
	// structure with the color values.
	if format.TrueColor {
		if err := binary.Write(&buf, binary.BigEndian, format.RedMax); err != nil {
			return nil, err
		}

		if err := binary.Write(&buf, binary.BigEndian, format.GreenMax); err != nil {
			return nil, err
		}

		if err := binary.Write(&buf, binary.BigEndian, format.BlueMax); err != nil {
			return nil, err
		}

		if err := binary.Write(&buf, binary.BigEndian, format.RedShift); err != nil {
			return nil, err
		}

		if err := binary.Write(&buf, binary.BigEndian, format.GreenShift); err != nil {
			return nil, err
		}

		if err := binary.Write(&buf, binary.BigEndian, format.BlueShift); err != nil {
			return nil, err
		}
	}

	return buf.Bytes()[0:16], nil
}
