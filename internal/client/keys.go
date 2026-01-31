package client

const (
	KEY_NULL byte = 0x00

	KEY_CTRL_A byte = 0x01
	KEY_CTRL_B byte = 0x02
	KEY_CTRL_C byte = 0x03 // interrupt
	KEY_CTRL_D byte = 0x04 // EOF
	KEY_CTRL_E byte = 0x05
	KEY_CTRL_F byte = 0x06
	KEY_CTRL_G byte = 0x07 // bell

	KEY_BACKSPACE byte = 0x7F // DEL (most terminals)

	KEY_TAB   byte = 0x09
	KEY_ENTER byte = 0x0D // carriage return
	KEY_ESC   byte = 0x1B
)

// Printable ASCII range
// 0x20 (space) → 0x7E (~)

func isPrintable(b byte) bool {
	return b >= 0x20 && b <= 0x7E
}

var (
	KEY_UP    = []byte{0x1B, 0x5B, 'A'}
	KEY_DOWN  = []byte{0x1B, 0x5B, 'B'}
	KEY_RIGHT = []byte{0x1B, 0x5B, 'C'}
	KEY_LEFT  = []byte{0x1B, 0x5B, 'D'}
	KEY_HOME  = []byte{0x1B, 0x5B, 'H'}
	KEY_END   = []byte{0x1B, 0x5B, 'F'}

	KEY_DELETE    = []byte{0x1B, 0x5B, '3', '~'}
	KEY_PAGE_UP   = []byte{0x1B, 0x5B, '5', '~'}
	KEY_PAGE_DOWN = []byte{0x1B, 0x5B, '6', '~'}
)
