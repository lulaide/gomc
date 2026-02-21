package nbt

import (
	"errors"
	"unicode/utf16"
)

func encodeModifiedUTF8(s string) ([]byte, error) {
	units := utf16.Encode([]rune(s))
	out := make([]byte, 0, len(units)*3)

	for _, u := range units {
		switch {
		case u >= 0x0001 && u <= 0x007F:
			out = append(out, byte(u))
		case u <= 0x07FF:
			out = append(out,
				byte(0xC0|((u>>6)&0x1F)),
				byte(0x80|(u&0x3F)),
			)
		default:
			out = append(out,
				byte(0xE0|((u>>12)&0x0F)),
				byte(0x80|((u>>6)&0x3F)),
				byte(0x80|(u&0x3F)),
			)
		}
	}

	if len(out) > 65535 {
		return nil, errors.New("modified UTF-8 string too long")
	}
	return out, nil
}

func decodeModifiedUTF8(data []byte) (string, error) {
	units := make([]uint16, 0, len(data))
	for i := 0; i < len(data); {
		c := data[i]
		switch {
		case c&0x80 == 0:
			units = append(units, uint16(c))
			i++
		case c&0xE0 == 0xC0:
			if i+1 >= len(data) {
				return "", errors.New("invalid modified UTF-8: truncated 2-byte sequence")
			}
			c2 := data[i+1]
			if c2&0xC0 != 0x80 {
				return "", errors.New("invalid modified UTF-8: invalid continuation byte")
			}
			u := uint16(c&0x1F)<<6 | uint16(c2&0x3F)
			units = append(units, u)
			i += 2
		case c&0xF0 == 0xE0:
			if i+2 >= len(data) {
				return "", errors.New("invalid modified UTF-8: truncated 3-byte sequence")
			}
			c2 := data[i+1]
			c3 := data[i+2]
			if c2&0xC0 != 0x80 || c3&0xC0 != 0x80 {
				return "", errors.New("invalid modified UTF-8: invalid continuation bytes")
			}
			u := uint16(c&0x0F)<<12 | uint16(c2&0x3F)<<6 | uint16(c3&0x3F)
			units = append(units, u)
			i += 3
		default:
			return "", errors.New("invalid modified UTF-8: unsupported leading byte")
		}
	}

	return string(utf16.Decode(units)), nil
}
