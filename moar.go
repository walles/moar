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

	if stdinIsRedirected && !stdoutIsRedirected {
		// FIXME: Display input pipe contents
		os.Exit(0)
	}

	// FIXME: Support --help

	// FIXME: Support --version

	if len(os.Args) != 2 {
		// FIXME: Improve this message
		fmt.Fprintf(os.Stderr, "ERROR: Expected exactly one parameter, got: %v\n", os.Args[1:])
		os.Exit(1)
	}

	if stdoutIsRedirected {
		input, err := os.Open(os.Args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
			os.Exit(1)
		}
		defer input.Close()

		// Copy input file to redirected stdout
		io.Copy(os.Stdout, input)
		os.Exit(0)
	}

	// FIXME: Display the input file contents
}
