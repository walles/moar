package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/alecthomas/chroma"
	"github.com/alecthomas/chroma/formatters"
	"github.com/alecthomas/chroma/styles"
	log "github.com/sirupsen/logrus"
	"golang.org/x/term"

	"github.com/walles/moar/m"
	"github.com/walles/moar/twin"
)

var versionString = "Should be set when building, please use build.sh to build"

func printUsage(output io.Writer) {
	// This controls where PrintDefaults() prints, see below
	flag.CommandLine.SetOutput(output)

	// FIXME: Log if any of these printouts fail?

	_, _ = fmt.Fprintln(output, "Usage:")
	_, _ = fmt.Fprintln(output, "  moar [options] <file>")
	_, _ = fmt.Fprintln(output, "  ... | moar")
	_, _ = fmt.Fprintln(output, "  moar < file")
	_, _ = fmt.Fprintln(output)

	flag.PrintDefaults()

	moarPath, err := filepath.Abs(os.Args[0])
	if err == nil {
		pagerValue, err := filepath.Abs(os.Getenv("PAGER"))
		if err != nil {
			pagerValue = ""
		}
		if pagerValue != moarPath {
			// We're not the default pager
			_, _ = fmt.Fprintln(output)
			_, _ = fmt.Fprintln(output, "To make Moar your default pager, put the following line in")
			_, _ = fmt.Fprintln(output, "your .bashrc or .bash_profile and it will be default in all")
			_, _ = fmt.Fprintln(output, "new terminal windows:")
			_, _ = fmt.Fprintf(output, "   export PAGER=%s\n", moarPath)
		}
	} else {
		log.Warn("Unable to find moar binary ", err)
	}
}

// printProblemsHeader prints bug reporting information to stderr
func printProblemsHeader() {
	fmt.Fprintln(os.Stderr, "Please post the following report at <https://github.com/walles/moar/issues>,")
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
}

func parseStyleOption(styleOption string) chroma.Style {
	style, ok := styles.Registry[styleOption]
	if !ok {
		fmt.Fprintf(os.Stderr,
			"ERROR: Unrecognized style \"%s\", pick a style from here: https://xyproto.github.io/splash/docs/longer/all.html\n",
			styleOption)
		fmt.Fprintln(os.Stderr)
		printUsage(os.Stderr)

		os.Exit(1)
	}

	return *style
}

func parseColorsOption(colorsOption string) chroma.Formatter {
	switch strings.ToUpper(colorsOption) {
	case "8":
		return formatters.TTY8
	case "16":
		return formatters.TTY16
	case "256":
		return formatters.TTY256
	case "16M":
		return formatters.TTY16m
	}

	fmt.Fprintf(os.Stderr, "ERROR: Invalid color count \"%s\", valid counts are 8, 16, 256 or 16M.\n", colorsOption)
	fmt.Fprintln(os.Stderr)
	printUsage(os.Stderr)

	os.Exit(1)
	panic("We just did os.Exit(), why are we still executing?")
}

func main() {
	// FIXME: If we get a CTRL-C, get terminal back into a useful state before terminating

	defer func() {
		err := recover()
		if err == nil {
			return
		}

		printProblemsHeader()
		panic(err)
	}()

	flag.Usage = func() {
		printUsage(os.Stdout)
	}
	printVersion := flag.Bool("version", false, "Prints the moar version number")
	debug := flag.Bool("debug", false, "Print debug logs after exiting")
	trace := flag.Bool("trace", false, "Print trace logs after exiting")
	styleOption := flag.String("style", "native", "Highlighting style from https://xyproto.github.io/splash/docs/longer/all.html")
	colorsOption := flag.String("colors", "16M", "Highlighting palette size: 8, 16, 256, 16M")

	flag.Parse()
	if *printVersion {
		fmt.Println(versionString)
		os.Exit(0)
	}

	style := parseStyleOption(*styleOption)
	formatter := parseColorsOption(*colorsOption)

	log.SetLevel(log.InfoLevel)
	if *trace {
		log.SetLevel(log.TraceLevel)
	} else if *debug {
		log.SetLevel(log.DebugLevel)
	}

	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: time.RFC3339Nano,
	})

	if len(flag.Args()) > 1 {
		fmt.Fprintln(os.Stderr, "ERROR: Expected exactly one filename, or data piped from stdin, got: ", flag.Args())
		fmt.Fprintln(os.Stderr)
		printUsage(os.Stderr)

		os.Exit(1)
	}

	stdinIsRedirected := !term.IsTerminal(int(os.Stdin.Fd()))
	stdoutIsRedirected := !term.IsTerminal(int(os.Stdout.Fd()))
	var inputFilename *string
	if len(flag.Args()) == 1 {
		word := flag.Arg(0)
		inputFilename = &word
	}

	if inputFilename == nil && !stdinIsRedirected {
		fmt.Fprintln(os.Stderr, "ERROR: Filename or input pipe required")
		fmt.Fprintln(os.Stderr, "")
		printUsage(os.Stderr)
		os.Exit(1)
	}

	if inputFilename != nil && stdoutIsRedirected {
		// Pump file to stdout.
		//
		// If we get both redirected stdin and an input filename, we must prefer
		// to copy the file, because that's how less works.
		inputFile, err := os.Open(*inputFilename)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERROR: Failed to open", inputFile, ": ")
			os.Exit(1)
		}

		_, err = io.Copy(os.Stdout, inputFile)
		if err != nil {
			log.Fatal("Failed to copy ", inputFilename, " to stdout: ", err)
		}
		os.Exit(0)
	}

	if stdinIsRedirected && stdoutIsRedirected {
		// Must be done after trying to pump the input filename to stdout to be
		// compatible with less, see above.
		_, err := io.Copy(os.Stdout, os.Stdin)
		if err != nil {
			log.Fatal("Failed to copy stdin to stdout: ", err)
		}
		os.Exit(0)
	}

	// INVARIANT: At this point, stdoutIsRedirected is false and we should
	// proceed with paging.

	if stdinIsRedirected {
		// Display input pipe contents
		reader := m.NewReaderFromStream("", os.Stdin)
		startPaging(reader)
		return
	}

	// Display the input file contents
	reader, err := m.NewReaderFromFilename(*inputFilename, style, formatter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
	startPaging(reader)
}

func startPaging(reader *m.Reader) {
	screen, e := twin.NewScreen()
	if e != nil {
		panic(e)
	}

	var loglines strings.Builder
	defer func() {
		// Restore screen...
		screen.Close()

		// ... before printing panic() output, otherwise the output will have
		// broken linefeeds and be hard to follow.
		if err := recover(); err != nil {
			panic(err)
		}

		if len(loglines.String()) > 0 {
			printProblemsHeader()

			// FIXME: Don't print duplicate log messages more than once,
			// maybe invent our own logger for this?
			fmt.Fprintf(os.Stderr, "%s", loglines.String())
			os.Exit(1)
		}
	}()

	log.SetOutput(&loglines)
	m.NewPager(reader).StartPaging(screen)
}
