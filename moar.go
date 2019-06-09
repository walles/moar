package main

import (
	"io"
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	// If stdin is a non-TTY and stdout is a non-TTY, just pump
	stdinIsRedirected := !terminal.IsTerminal(int(os.Stdin.Fd()))
	stdoutIsRedirected := !terminal.IsTerminal(int(os.Stdout.Fd()))
	if stdinIsRedirected && stdoutIsRedirected {
		io.Copy(os.Stdout, os.Stdin)
	}

	// FIXME: If first arg is a file name and stdout is a non-TTY, pump file onto stdout

}
