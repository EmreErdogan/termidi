package main

import (
	"os"

	"golang.org/x/sys/windows"
)

// Windows consoles only interpret ANSI escape sequences once virtual terminal
// processing is enabled.
func init() {
	h := windows.Handle(os.Stdout.Fd())
	var mode uint32
	if windows.GetConsoleMode(h, &mode) == nil {
		windows.SetConsoleMode(h, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	}
}
