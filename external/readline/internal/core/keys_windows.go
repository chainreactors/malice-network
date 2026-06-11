//go:build windows
// +build windows

package core

import (
	"errors"
	"io"
	"os"
	"time"
	"unsafe"

	"github.com/reeflective/readline/inputrc"
	"github.com/reeflective/readline/internal/term"
)

// Windows-specific special key codes.
const (
	VK_CANCEL   = 0x03
	VK_BACK     = 0x08
	VK_TAB      = 0x09
	VK_RETURN   = 0x0D
	VK_SHIFT    = 0x10
	VK_CONTROL  = 0x11
	VK_MENU     = 0x12
	VK_ESCAPE   = 0x1B
	VK_LEFT     = 0x25
	VK_UP       = 0x26
	VK_RIGHT    = 0x27
	VK_DOWN     = 0x28
	VK_DELETE   = 0x2E
	VK_LSHIFT   = 0xA0
	VK_RSHIFT   = 0xA1
	VK_LCONTROL = 0xA2
	VK_RCONTROL = 0xA3
	VK_SNAPSHOT = 0x2C
	VK_INSERT   = 0x2D
	VK_HOME     = 0x24
	VK_END      = 0x23
	VK_PRIOR    = 0x21
	VK_NEXT     = 0x22
)

// Use an undefined Virtual Key sequence to pass
// Windows terminal resize events from the reader.
const (
	WINDOWS_RESIZE = 0x07
)

const (
	charTab       = 9
	charCtrlH     = 8
	charBackspace = 127
)

// dwControlKeyState flags from Windows Console API.
const (
	_RIGHT_ALT_PRESSED  = 0x0001
	_LEFT_ALT_PRESSED   = 0x0002
	_RIGHT_CTRL_PRESSED = 0x0004
	_LEFT_CTRL_PRESSED  = 0x0008
	_SHIFT_PRESSED      = 0x0010
)

func init() {
	Stdin = newRawReader()
}

// readInputFiltered on Windows needs to check for terminal resize events.
func (k *Keys) readInputFiltered() (keys []byte, err error) {
	for {
		// Start reading from os.Stdin in the background.
		// We will either read keys from user, or an EOF
		// send by ourselves, because we pause reading.
		buf := make([]byte, keyScanBufSize)

		read, err := Stdin.Read(buf)
		if err != nil && errors.Is(err, io.EOF) {
			return keys, err
		}

		input := buf[:read]

		// On Windows, windows resize events are sent through stdin,
		// so if one is detected, send it back to the display engine.
		if len(input) == 1 && input[0] == WINDOWS_RESIZE {
			k.resize <- true
			continue
		}

		// Always attempt to extract cursor position info.
		// If found, strip it and keep the remaining keys.
		cursor, keys := k.extractCursorPos(input)

		if len(cursor) > 0 {
			k.cursor <- cursor
		}

		return keys, nil
	}
}

// rawReader translates Windows input to ANSI sequences,
// to provide the same behavior as Unix terminals.
type rawReader struct{}

// newRawReader returns a new rawReader for Windows.
func newRawReader() *rawReader {
	return new(rawReader)
}

// isCtrl returns true if Ctrl is pressed in this event's dwControlKeyState.
func isCtrl(state dword) bool {
	return state&(_LEFT_CTRL_PRESSED|_RIGHT_CTRL_PRESSED) != 0
}

// isAlt returns true if Alt is pressed in this event's dwControlKeyState.
func isAlt(state dword) bool {
	return state&(_LEFT_ALT_PRESSED|_RIGHT_ALT_PRESSED) != 0
}

// isShift returns true if Shift is pressed in this event's dwControlKeyState.
func isShift(state dword) bool {
	return state&_SHIFT_PRESSED != 0
}

// Read reads input record from stdin on Windows.
// It keeps reading until it gets a key event.
func (r *rawReader) Read(buf []byte) (int, error) {
	ir := new(_INPUT_RECORD)
	var read int
	var err error

next:
	// ReadConsoleInputW reads input record from stdin.
	err = kernel.ReadConsoleInputW(stdin,
		uintptr(unsafe.Pointer(ir)),
		1,
		uintptr(unsafe.Pointer(&read)),
	)
	if err != nil {
		return 0, err
	}

	// Skip focus events.
	if ir.EventType == EVENT_FOCUS {
		goto next
	}

	// Keep resize events for the display engine to use.
	if ir.EventType == EVENT_WINDOW_BUFFER_SIZE {
		return r.write(buf, WINDOWS_RESIZE)
	}

	if ir.EventType != EVENT_KEY {
		goto next
	}

	ker := (*_KEY_EVENT_RECORD)(unsafe.Pointer(&ir.Event[0]))

	// Skip key-up events.
	if ker.bKeyDown == 0 {
		goto next
	}

	// Skip standalone modifier key presses.
	switch ker.wVirtualKeyCode {
	case VK_CONTROL, VK_LCONTROL, VK_RCONTROL,
		VK_MENU,
		VK_SHIFT, VK_LSHIFT, VK_RSHIFT:
		goto next
	}

	// Use per-event dwControlKeyState for reliable modifier detection.
	// This avoids stale modifier state when the terminal synthesizes
	// key events (e.g., bracketed paste after Ctrl+V interception).
	ctrlKey := isCtrl(ker.dwControlKeyState)
	altKey := isAlt(ker.dwControlKeyState)
	shiftKey := isShift(ker.dwControlKeyState)

	// Keypad, special and arrow keys (unicodeChar == 0).
	if ker.unicodeChar == 0 {
		if modifiers, target := r.translateSeq(ker, ctrlKey, altKey, shiftKey); target != 0 {
			return r.writeEsc(buf, append(modifiers, target)...)
		}
		goto next
	}

	{
		char := rune(ker.unicodeChar)

		// Encode keys with modifiers.
		switch {
		case shiftKey && char == charTab:
			return r.writeEsc(buf, 91, 90)
		case ctrlKey && char == charBackspace:
			char = charCtrlH
		case !ctrlKey && char == charCtrlH:
			char = charBackspace
		case ctrlKey:
			char = inputrc.Encontrol(char)
		case altKey:
			char = inputrc.Enmeta(char)
		}

		return r.write(buf, char)
	}
}

// Close is a stub to satisfy io.Closer.
func (r *rawReader) Close() error {
	return nil
}

func (r *rawReader) writeEsc(b []byte, char ...rune) (int, error) {
	b[0] = byte(inputrc.Esc)
	n := copy(b[1:], []byte(string(char)))
	return n + 1, nil
}

func (r *rawReader) write(b []byte, char ...rune) (int, error) {
	n := copy(b, []byte(string(char)))
	return n, nil
}

func (r *rawReader) translateSeq(ker *_KEY_EVENT_RECORD, ctrlKey, altKey, shiftKey bool) (modifiers []rune, target rune) {
	// Encode keys with modifiers by default.
	modifiers = append(modifiers, 91)

	// Modifiers add a default sequence, which is the good sequence for arrow keys by default.
	switch {
	case ctrlKey:
		modifiers = append(modifiers, 49, 59, 53)
	case altKey:
		modifiers = append(modifiers, 49, 59, 51)
	case shiftKey:
		modifiers = append(modifiers, 49, 59, 50)
	}

	changeModifiers := func(swap rune, pos int) {
		if len(modifiers) > pos-1 && pos > 0 {
			modifiers[pos] = swap
		} else {
			modifiers = append(modifiers, swap)
		}
	}

	// Now we handle the target key.
	switch ker.wVirtualKeyCode {
	// Keypad & arrow keys
	case VK_LEFT:
		target = 68
	case VK_RIGHT:
		target = 67
	case VK_UP:
		target = 65
	case VK_DOWN:
		target = 66
	case VK_HOME:
		target = 72
	case VK_END:
		target = 70

	// Other special keys, with effects on modifiers.
	case VK_SNAPSHOT:
	case VK_INSERT:
		changeModifiers(50, 2)
		target = 126
	case VK_DELETE:
		changeModifiers(51, 2)
		target = 126
	case VK_PRIOR:
		changeModifiers(53, 2)
		target = 126
	case VK_NEXT:
		changeModifiers(54, 2)
		target = 126
	}

	return
}

// GetTerminalResize sends booleans over a channel to notify resize events on Windows.
// This functions uses the keys reader because on Windows, resize events are sent through
// stdin, not with syscalls like unix's syscall.SIGWINCH.
func GetTerminalResize(keys *Keys) <-chan bool {
	keys.resize = make(chan bool, 1)
	prevWidth, prevHeight, _ := term.GetSize(int(os.Stdout.Fd()))
	go func() {
		for {
			width, height, err := term.GetSize(int(os.Stdout.Fd()))
			if err != nil {
				break
			}

			if width != prevWidth || height != prevHeight {
				prevWidth, prevHeight = width, height
				//fmt.Println("windows resize")
				keys.resize <- true
			}
			time.Sleep(500 * time.Millisecond)
		}
	}()
	return keys.resize
}
