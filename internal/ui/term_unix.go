//go:build !windows

package ui

import (
	"os"

	"golang.org/x/sys/unix"
)

var origTermios *unix.Termios

func EnableRawMode() error {
	fd := int(os.Stdin.Fd())
	t, err := unix.IoctlGetTermios(fd, ioctlGetTermios)
	if err != nil {
		return err
	}
	origTermios = new(unix.Termios)
	*origTermios = *t

	// Only disable canonical mode and echo.
	// Keep OPOST so \n produces \r\n, and keep ISIG so Ctrl+C works.
	t.Lflag &^= unix.ICANON | unix.ECHO
	t.Cc[unix.VMIN] = 1
	t.Cc[unix.VTIME] = 0

	return unix.IoctlSetTermios(fd, ioctlSetTermios, t)
}

func DisableRawMode() {
	if origTermios == nil {
		return
	}
	fd := int(os.Stdin.Fd())
	_ = unix.IoctlSetTermios(fd, ioctlSetTermios, origTermios)
	origTermios = nil
}

func ReadByte() (byte, error) {
	var buf [1]byte
	_, err := os.Stdin.Read(buf[:])
	return buf[0], err
}
