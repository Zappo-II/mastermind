//go:build windows

package ui

import (
	"os"

	"golang.org/x/sys/windows"
)

var (
	origInMode  uint32
	origOutMode uint32
	inHandle    windows.Handle
	outHandle   windows.Handle
	saved       bool
)

func EnableRawMode() error {
	var err error
	inHandle, err = windows.GetStdHandle(windows.STD_INPUT_HANDLE)
	if err != nil {
		return err
	}
	outHandle, err = windows.GetStdHandle(windows.STD_OUTPUT_HANDLE)
	if err != nil {
		return err
	}

	// Save original modes
	if err := windows.GetConsoleMode(inHandle, &origInMode); err != nil {
		return err
	}
	if err := windows.GetConsoleMode(outHandle, &origOutMode); err != nil {
		return err
	}
	saved = true

	// Disable line input and echo (like ICANON and ECHO on Unix)
	inMode := origInMode
	inMode &^= windows.ENABLE_LINE_INPUT | windows.ENABLE_ECHO_INPUT
	if err := windows.SetConsoleMode(inHandle, inMode); err != nil {
		return err
	}

	// Enable ANSI escape sequence processing on stdout
	outMode := origOutMode | windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING
	if err := windows.SetConsoleMode(outHandle, outMode); err != nil {
		// Non-fatal: old Windows versions don't support VT processing.
		// The game will look broken but won't crash.
		_ = err
	}

	return nil
}

func DisableRawMode() {
	if !saved {
		return
	}
	windows.SetConsoleMode(inHandle, origInMode)
	windows.SetConsoleMode(outHandle, origOutMode)
	saved = false
}

func ReadByte() (byte, error) {
	var buf [1]byte
	_, err := os.Stdin.Read(buf[:])
	return buf[0], err
}
