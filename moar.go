package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/gdamore/tcell"
	"github.com/walles/moar/m"

	"golang.org/x/crypto/ssh/terminal"
)

var versionString = "Should be set when building, please use build.sh to build"

func main() {
	// FIXME: If we get a CTRL-C, get terminal back into a useful state before terminating

	defer func() {
		err := recover()
		if err == nil {
			return
		}

		// On any panic or warnings, also print system info and how to report bugs
		fmt.Fprintln(os.Stderr, "Please post the following crash report at <https://github.com/walles/moar/issues>,")
		fmt.Fprintln(os.Stderr, "or e-mail it to johan.walles@gmail.com.")
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "Version:", versionString)
		fmt.Fprintln(os.Stderr, "LANG   :", os.Getenv("LANG"))
		fmt.Fprintln(os.Stderr, "TERM   :", os.Getenv("TERM"))
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, "GOOS    :", runtime.GOOS)
		fmt.Fprintln(os.Stderr, "GOARCH  :", runtime.GOARCH)
		fmt.Fprintln(os.Stderr, "Compiler:", runtime.Compiler)
		fmt.Fprintln(os.Stderr, "NumCPU  :", runtime.NumCPU())

		fmt.Fprintln(os.Stderr)

		panic(err)
	}()

	printVersion := flag.Bool("version", false, "Prints the moar version number")

	// FIXME: Support --help

	// FIXME: Support --no-highlight

	flag.Parse()
	if *printVersion {
		fmt.Println(versionString)
		os.Exit(0)
	}

	stdinIsRedirected := !terminal.IsTerminal(int(os.Stdin.Fd()))
	stdoutIsRedirected := !terminal.IsTerminal(int(os.Stdout.Fd()))
	if stdinIsRedirected && stdoutIsRedirected {
		io.Copy(os.Stdout, os.Stdin)
		os.Exit(0)
	}

	if stdinIsRedirected && !stdoutIsRedirected {
		// Display input pipe contents
		reader := m.NewReaderFromStream(os.Stdin, nil)
		_StartPaging(reader)
		return
	}

	if len(flag.Args()) != 1 {
		// FIXME: Improve this message
		fmt.Fprintln(os.Stderr, "ERROR: Expected exactly one filename, got: ", flag.Args())

		// FIXME: Print full usage here

		os.Exit(1)
	}

	if stdoutIsRedirected {
		// Pump from file by given name onto stdout which is redirected
		input, err := os.Open(flag.Arg(0))
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
			os.Exit(1)
		}
		defer input.Close()

		// Copy input file to redirected stdout
		io.Copy(os.Stdout, input)
		os.Exit(0)
	}

	// Display the input file contents
	reader, err := m.NewReaderFromFilename(flag.Arg(0))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
	_StartPaging(reader)
}

func _StartPaging(reader *m.Reader) {
	screen, e := tcell.NewScreen()
	if e != nil {
		panic(e)
	}

	var loglines strings.Builder
	defer func() {
		// Restore screen...
		screen.Fini()

		// ... before printing panic() output, otherwise the output will have
		// broken linefeeds and be hard to follow.
		if err := recover(); err != nil {
			panic(err)
		}

		if len(loglines.String()) > 0 {
			// FIXME: Don't print duplicate log messages more than once,
			// maybe invent our own logger for this?
			fmt.Fprintf(os.Stderr, "%s", loglines.String())
			os.Exit(1)
		}
	}()

	logger := log.New(&loglines, "", 0)
	m.NewPager(reader).StartPaging(logger, screen)
}
