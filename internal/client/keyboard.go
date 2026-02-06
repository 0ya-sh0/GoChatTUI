package client

import (
	"os"
	"time"
)

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

const (
	KEY_TYPE_PRINTABLE = iota
	KEY_TYPE_CTRL_C
	KEY_TYPE_ESC
	KEY_TYPE_UP_ARROW
	KEY_TYPE_DOWN_ARROW
	KEY_TYPE_LEFT_ARROW
	KEY_TYPE_RIGHT_ARROW
	KEY_TYPE_ENTER
	KEY_TYPE_BACKSPACE
	KEY_TYPE_UNKNOWN
)

type KeyType int

type EventKeyPress struct {
	KeyType KeyType
	Char    byte
}

func listenKeyEvents(events chan EventKeyPress) {
	for {
		event, err := readKey()
		if err != nil {
			close(events)
			break
		}
		events <- event
	}
}

func readKey() (EventKeyPress, error) {
	bt := make([]byte, 1)
	_, err := os.Stdin.Read(bt)
	if err != nil {
		return EventKeyPress{}, err
	}

	if isPrintable(bt[0]) {
		return EventKeyPress{
			KeyType: KEY_TYPE_PRINTABLE,
			Char:    bt[0],
		}, nil
	}

	if bt[0] == KEY_CTRL_C {
		return EventKeyPress{
			KeyType: KEY_TYPE_CTRL_C,
			Char:    bt[0],
		}, nil
	}

	if bt[0] == KEY_ENTER {
		return EventKeyPress{
			KeyType: KEY_TYPE_ENTER,
			Char:    bt[0],
		}, nil
	}

	if bt[0] == KEY_BACKSPACE {
		return EventKeyPress{
			KeyType: KEY_TYPE_BACKSPACE,
			Char:    bt[0],
		}, nil
	}

	if bt[0] == KEY_ESC {
		os.Stdin.SetReadDeadline(time.Now().Add(20 * time.Millisecond))
		defer os.Stdin.SetReadDeadline(time.Time{})

		seq := []byte{KEY_ESC}

		for {
			_, err := os.Stdin.Read(bt)
			if err != nil {
				return EventKeyPress{
					KeyType: KEY_TYPE_ESC,
					Char:    KEY_ESC,
				}, nil
			}
			seq = append(seq, bt[0])
			if len(seq) >= 3 {
				if string(seq) == string(KEY_UP) {
					return EventKeyPress{
						KeyType: KEY_TYPE_UP_ARROW,
					}, nil
				}

				if string(seq) == string(KEY_DOWN) {
					return EventKeyPress{
						KeyType: KEY_TYPE_DOWN_ARROW,
					}, nil
				}

				if string(seq) == string(KEY_LEFT) {
					return EventKeyPress{
						KeyType: KEY_TYPE_LEFT_ARROW,
					}, nil
				}

				if string(seq) == string(KEY_RIGHT) {
					return EventKeyPress{
						KeyType: KEY_TYPE_RIGHT_ARROW,
					}, nil
				}
				return EventKeyPress{
					KeyType: KEY_TYPE_UNKNOWN,
				}, nil
			}
		}
	}

	return EventKeyPress{
		KeyType: KEY_TYPE_UNKNOWN,
		Char:    bt[0],
	}, nil
}
