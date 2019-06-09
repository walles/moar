package main

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	stdinIsRedirected := !terminal.IsTerminal(int(os.Stdin.Fd()))
	stdoutIsRedirected := !terminal.IsTerminal(int(os.Stdout.Fd()))
	if stdinIsRedirected && stdoutIsRedirected {
		io.Copy(os.Stdout, os.Stdin)
		os.Exit(0)
	}

	// FIXME: Support --help

	// FIXME: Support --version

	if len(os.Args) != 2 {
		// FIXME: Improve this message
		fmt.Fprintf(os.Stderr, "ERROR: Expected exactly one parameter, got: %v\n", os.Args[1:])
		os.Exit(1)
	}

	// FIXME: If first arg is a file name and stdout is a non-TTY, pump file onto stdout
	if stdoutIsRedirected {
		input, err := os.Open(os.Args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
			os.Exit(1)
		}

		io.Copy(os.Stdout, input)
	}
}
