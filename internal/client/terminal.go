package client

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/term"
)

const (
	ESC = "\x1b"

	// Cursor movement
	CursorHome    = ESC + "[H"
	CursorHide    = ESC + "[?25l"
	CursorShow    = ESC + "[?25h"
	CursorSave    = ESC + "7"
	CursorRestore = ESC + "8"

	// Screen clearing
	ClearScreen    = ESC + "[2J"
	ClearLine      = ESC + "[2K"
	ClearLineRight = ESC + "[0K"
	ClearLineLeft  = ESC + "[1K"

	// Cursor positioning (use fmt.Sprintf)
	CursorPos = ESC + "[%d;%dH" // row, col (1-based)

	// Scrolling
	ScrollUp   = ESC + "[S"
	ScrollDown = ESC + "[T"

	// Text styles
	Reset     = ESC + "[0m"
	Bold      = ESC + "[1m"
	Dim       = ESC + "[2m"
	Underline = ESC + "[4m"
	Reverse   = ESC + "[7m"

	// Foreground colors
	FgBlack   = ESC + "[30m"
	FgRed     = ESC + "[31m"
	FgGreen   = ESC + "[32m"
	FgYellow  = ESC + "[33m"
	FgBlue    = ESC + "[34m"
	FgMagenta = ESC + "[35m"
	FgCyan    = ESC + "[36m"
	FgWhite   = ESC + "[37m"

	// Background colors
	BgBlack   = ESC + "[40m"
	BgRed     = ESC + "[41m"
	BgGreen   = ESC + "[42m"
	BgYellow  = ESC + "[43m"
	BgBlue    = ESC + "[44m"
	BgMagenta = ESC + "[45m"
	BgCyan    = ESC + "[46m"
	BgWhite   = ESC + "[47m"
)

const (
	ANSI_ENTER_ALT_SCREEN = "\x1b[?1049h"
	ANSI_EXIT_ALT_SCREEN  = "\x1b[?1049l"
)

func listenResizeEvents(events chan int) {
	_, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		close(events)
		return
	}
	for {
		time.Sleep(time.Millisecond * 500)
		_, newHeight, err := term.GetSize(int(os.Stdin.Fd()))
		if err != nil {
			close(events)
			return
		}
		if newHeight != height {
			height = newHeight
			events <- height
		}
	}
}

var oldState *term.State

func SetupTerminal() error {
	state, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	oldState = state
	fmt.Print(ANSI_ENTER_ALT_SCREEN)
	return nil
}

func RestoreTerminal() {
	fmt.Print(ClearScreen, CursorHome, CursorShow)
	fmt.Print(ANSI_EXIT_ALT_SCREEN)
	term.Restore(int(os.Stdin.Fd()), oldState)
}
