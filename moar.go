package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	log "github.com/sirupsen/logrus"
	"golang.org/x/term"

	"github.com/walles/moar/m"
	"github.com/walles/moar/m/linenumbers"
	"github.com/walles/moar/m/textstyles"
	"github.com/walles/moar/twin"
)

const defaultDarkTheme = "native"

// I decided on a light theme by doing this:
//
//	wc -l ../chroma/styles/*.xml|sort|cut -d/ -f4|grep xml|xargs -I XXX grep -Hi background ../chroma/styles/XXX
//
// Then I picked tango because it has a lot of lines, a bright background
// and I like the looks of it.
const defaultLightTheme = "tango"

var versionString = ""

// printProblemsHeader prints bug reporting information to stderr
func printProblemsHeader() {
	fmt.Fprintln(os.Stderr, "Please post the following report at <https://github.com/walles/moar/issues>,")
	fmt.Fprintln(os.Stderr, "or e-mail it to johan.walles@gmail.com.")
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "Version:", getVersion())
	fmt.Fprintln(os.Stderr, "LANG   :", os.Getenv("LANG"))
	fmt.Fprintln(os.Stderr, "TERM   :", os.Getenv("TERM"))
	fmt.Fprintln(os.Stderr, "MOAR   :", os.Getenv("MOAR"))
	fmt.Fprintln(os.Stderr, "EDITOR :", os.Getenv("EDITOR"))
	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "GOOS    :", runtime.GOOS)
	fmt.Fprintln(os.Stderr, "GOARCH  :", runtime.GOARCH)
	fmt.Fprintln(os.Stderr, "Compiler:", runtime.Compiler)
	fmt.Fprintln(os.Stderr, "NumCPU  :", runtime.NumCPU())
}

func parseLexerOption(lexerOption string) (chroma.Lexer, error) {
	byMimeType := lexers.MatchMimeType(lexerOption)
	if byMimeType != nil {
		return byMimeType, nil
	}

	// Use Chroma's built-in fuzzy lexer picker
	lexer := lexers.Get(lexerOption)
	if lexer != nil {
		return lexer, nil
	}

	return nil, fmt.Errorf(
		"Look here for inspiration: https://github.com/alecthomas/chroma/tree/master/lexers/embedded",
	)
}

func parseStyleOption(styleOption string) (*chroma.Style, error) {
	style, ok := styles.Registry[styleOption]
	if !ok {
		return &chroma.Style{}, fmt.Errorf(
			"Pick a style from here: https://xyproto.github.io/splash/docs/longer/all.html")
	}

	return style, nil
}

func parseColorsOption(colorsOption string) (twin.ColorCount, error) {
	if strings.ToLower(colorsOption) == "auto" {
		colorsOption = "16M"
		if os.Getenv("COLORTERM") != "truecolor" && strings.Contains(os.Getenv("TERM"), "256") {
			// Covers "xterm-256color" as used by the macOS Terminal
			colorsOption = "256"
		}
	}

	switch strings.ToUpper(colorsOption) {
	case "8":
		return twin.ColorCount8, nil
	case "16":
		return twin.ColorCount16, nil
	case "256":
		return twin.ColorCount256, nil
	case "16M":
		return twin.ColorCount24bit, nil
	}

	var noColor twin.ColorCount
	return noColor, fmt.Errorf("Valid counts are 8, 16, 256, 16M or auto")
}

func parseStatusBarStyle(styleOption string) (m.StatusBarOption, error) {
	if styleOption == "inverse" {
		return m.STATUSBAR_STYLE_INVERSE, nil
	}
	if styleOption == "plain" {
		return m.STATUSBAR_STYLE_PLAIN, nil
	}
	if styleOption == "bold" {
		return m.STATUSBAR_STYLE_BOLD, nil
	}

	return 0, fmt.Errorf("Good ones are inverse, plain and bold")
}

func parseUnprintableStyle(styleOption string) (textstyles.UnprintableStyleT, error) {
	if styleOption == "highlight" {
		return textstyles.UnprintableStyleHighlight, nil
	}
	if styleOption == "whitespace" {
		return textstyles.UnprintableStyleWhitespace, nil
	}

	return 0, fmt.Errorf("Good ones are highlight or whitespace")
}

func parseScrollHint(scrollHint string) (twin.StyledRune, error) {
	scrollHint = strings.ReplaceAll(scrollHint, "ESC", "\x1b")
	hintAsLine := m.NewLine(scrollHint)
	parsedTokens := hintAsLine.HighlightedTokens("", nil, nil).StyledRunes
	if len(parsedTokens) == 1 {
		return parsedTokens[0], nil
	}

	return twin.StyledRune{}, fmt.Errorf("Expected exactly one (optionally highlighted) character. For example: 'ESC[2m…'")
}

func parseShiftAmount(shiftAmount string) (uint, error) {
	value, err := strconv.ParseUint(shiftAmount, 10, 32)
	if err != nil {
		return 0, err
	}

	if value < 1 {
		return 0, fmt.Errorf("Shift amount must be at least 1")
	}

	// Let's add an upper bound as well if / when requested

	return uint(value), nil
}

func parseMouseMode(mouseMode string) (twin.MouseMode, error) {
	switch mouseMode {
	case "auto":
		return twin.MouseModeAuto, nil
	case "select", "mark":
		return twin.MouseModeSelect, nil
	case "scroll":
		return twin.MouseModeScroll, nil
	}

	return twin.MouseModeAuto, fmt.Errorf("Valid modes are auto, select and scroll")
}

func pumpToStdout(inputFilenames ...string) error {
	if len(inputFilenames) > 0 {
		// If we get both redirected stdin and an input filenames, should only
		// copy the files and ignore stdin, because that's how less works.
		for _, inputFilename := range inputFilenames {
			inputFile, _, err := m.ZOpen(inputFilename)
			if err != nil {
				return fmt.Errorf("Failed to open %s: %w", inputFilename, err)
			}

			_, err = io.Copy(os.Stdout, inputFile)
			if err != nil {
				return fmt.Errorf("Failed to copy %s to stdout: %w", inputFilename, err)
			}
		}

		return nil
	}

	// No input filenames, pump stdin to stdout
	_, err := io.Copy(os.Stdout, os.Stdin)
	if err != nil {
		return fmt.Errorf("Failed to copy stdin to stdout: %w", err)
	}
	return nil
}

// Duplicate of m/reader.go:tryOpen
func tryOpen(filename string) error {
	// Try opening the file
	tryMe, err := os.Open(filename)
	if err != nil {
		return err
	}

	// Try reading a byte
	buffer := make([]byte, 1)
	_, err = tryMe.Read(buffer)

	if err != nil && err.Error() == "EOF" {
		// Empty file, this is fine
		err = nil
	}

	closeErr := tryMe.Close()
	if err == nil && closeErr != nil {
		// Everything worked up until Close(), report the Close() error
		return closeErr
	}

	return err
}

// Parses an argument like "+123" anywhere on the command line into a one-based
// line number, and returns the remaining args.
//
// Returns nil on no target line number specified.
func getTargetLineNumber(args []string) (*linenumbers.LineNumber, []string) {
	for i, arg := range args {
		if !strings.HasPrefix(arg, "+") {
			continue
		}

		lineNumber, err := strconv.ParseInt(arg[1:], 10, 32)
		if err != nil {
			// Let's pretend this is a file name
			continue
		}
		if lineNumber < 1 {
			// Pretend this is a file name
			continue
		}

		// Remove the target line number from the args
		//
		// Ref: https://stackoverflow.com/a/57213476/473672
		remainingArgs := make([]string, 0)
		remainingArgs = append(remainingArgs, args[:i]...)
		remainingArgs = append(remainingArgs, args[i+1:]...)

		returnMe := linenumbers.LineNumberFromOneBased(int(lineNumber))
		return &returnMe, remainingArgs
	}

	return nil, args
}

// On man pages, disable line numbers by default.
//
// Before paging, "man" first checks the terminal width and formats the man page
// to fit that width.
//
// Then, if moar adds line numbers, the rightmost part of the man page won't be
// visible.
//
// So we try to detect showing man pages, and in that case disable line numbers
// so that the rightmost part of the page is visible by default.
func noLineNumbersDefault() bool {
	if os.Getenv("MANPATH") != "" {
		// Set by "man" on macOS, skip line numbers in this case
		return true
	}

	if os.Getenv("MAN_PN") != "" {
		// Set by "man" on Ubuntu 22.04.4 when I tested it inside of Docker,
		// skip line numbers in this case
		return true
	}

	// Default to not skipping line numbers
	return false
}

// Return complete version when built with build.sh or fallback to module version (i.e. "go install")
func getVersion() string {
	if versionString != "" {
		return versionString
	}
	info, ok := debug.ReadBuildInfo()
	if ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "Should be set when building, please use build.sh to build"
}

// Can return a nil pager on --help or --version, or if pumping to stdout.
func pagerFromArgs(
	args []string,
	newScreen func(mouseMode twin.MouseMode, terminalColorCount twin.ColorCount) (twin.Screen, error),
	stdinIsRedirected bool,
	stdoutIsRedirected bool,
) (
	*m.Pager, twin.Screen, chroma.Style, *chroma.Formatter, error,
) {
	// FIXME: If we get a CTRL-C, get terminal back into a useful state before terminating

	flagSet := flag.NewFlagSet("",
		flag.ContinueOnError, // We want to do our own error handling
	)
	flagSet.SetOutput(io.Discard) // We want to do our own printing

	printVersion := flagSet.Bool("version", false, "Prints the moar version number")
	debug := flagSet.Bool("debug", false, "Print debug logs after exiting")
	trace := flagSet.Bool("trace", false, "Print trace logs after exiting")

	wrap := flagSet.Bool("wrap", false, "Wrap long lines")
	follow := flagSet.Bool("follow", false, "Follow piped input just like \"tail -f\"")
	styleOption := flagSetFunc(flagSet,
		"style", nil,
		"Highlighting `style` from https://xyproto.github.io/splash/docs/longer/all.html", parseStyleOption)
	lexer := flagSetFunc(flagSet,
		"lang", nil,
		"File contents, used for highlighting. Mime type or file extension (\"html\"). Default is to guess by filename.", parseLexerOption)

	defaultFormatter, err := parseColorsOption("auto")
	if err != nil {
		panic(fmt.Errorf("Failed parsing default formatter: %w", err))
	}
	terminalColorsCount := flagSetFunc(flagSet,
		"colors", defaultFormatter, "Highlighting palette size: 8, 16, 256, 16M, auto", parseColorsOption)

	noLineNumbers := flagSet.Bool("no-linenumbers", noLineNumbersDefault(), "Hide line numbers on startup, press left arrow key to show")
	noStatusBar := flagSet.Bool("no-statusbar", false, "Hide the status bar, toggle with '='")
	noReFormat := flagSet.Bool("no-reformat", false, "Never reformat the input (but keep highlighting)")
	quitIfOneScreen := flagSet.Bool("quit-if-one-screen", false, "Don't page if contents fits on one screen")
	noClearOnExit := flagSet.Bool("no-clear-on-exit", false, "Retain screen contents when exiting moar")
	statusBarStyle := flagSetFunc(flagSet, "statusbar", m.STATUSBAR_STYLE_INVERSE,
		"Status bar `style`: inverse, plain or bold", parseStatusBarStyle)
	unprintableStyle := flagSetFunc(flagSet, "render-unprintable", textstyles.UnprintableStyleHighlight,
		"How unprintable characters are rendered: highlight or whitespace", parseUnprintableStyle)
	scrollLeftHint := flagSetFunc(flagSet, "scroll-left-hint",
		twin.NewStyledRune('<', twin.StyleDefault.WithAttr(twin.AttrReverse)),
		"Shown when view can scroll left. One character with optional ANSI highlighting.", parseScrollHint)
	scrollRightHint := flagSetFunc(flagSet, "scroll-right-hint",
		twin.NewStyledRune('>', twin.StyleDefault.WithAttr(twin.AttrReverse)),
		"Shown when view can scroll right. One character with optional ANSI highlighting.", parseScrollHint)
	shift := flagSetFunc(flagSet, "shift", 16, "Horizontal scroll `amount` >=1, defaults to 16", parseShiftAmount)
	mouseMode := flagSetFunc(
		flagSet,
		"mousemode",
		twin.MouseModeAuto,
		"Mouse `mode`: auto, select or scroll: https://github.com/walles/moar/blob/master/MOUSE.md",
		parseMouseMode,
	)

	// Combine flags from environment and from command line
	flags := args[1:]
	moarEnv := strings.Trim(os.Getenv("MOAR"), " ")
	if len(moarEnv) > 0 {
		// FIXME: It would be nice if we could debug log that we're doing this,
		// but logging is not yet set up and depends on command line parameters.
		flags = append(strings.Fields(moarEnv), flags...)
	}

	targetLineNumber, remainingArgs := getTargetLineNumber(flags)

	err = flagSet.Parse(remainingArgs)
	if err != nil {
		if err == flag.ErrHelp {
			printUsage(flagSet, *terminalColorsCount)
			return nil, nil, chroma.Style{}, nil, nil
		}

		errorText := err.Error()
		if strings.HasPrefix(errorText, "invalid value") {
			errorText = strings.Replace(errorText, ": ", "\n\n", 1)
		}

		boldErrorMessage := "\x1b[1m" + errorText + "\x1b[m"
		fmt.Fprintln(os.Stderr, "ERROR:", boldErrorMessage)
		fmt.Fprintln(os.Stderr)
		printCommandline(os.Stderr)
		fmt.Fprintln(os.Stderr, "For help, run: \x1b[1mmoar --help\x1b[m")

		os.Exit(1)
	}

	if *printVersion {
		fmt.Println(getVersion())
		return nil, nil, chroma.Style{}, nil, nil
	}

	log.SetLevel(log.WarnLevel)
	if *trace {
		log.SetLevel(log.TraceLevel)
	} else if *debug {
		log.SetLevel(log.DebugLevel)
	}

	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: time.StampMicro,
	})

	if len(flagSet.Args()) > 1 && !stdoutIsRedirected {
		fmt.Fprintln(os.Stderr, "ERROR: Expected exactly one filename, or data piped from stdin")
		fmt.Fprintln(os.Stderr)
		printCommandline(os.Stderr)
		fmt.Fprintln(os.Stderr, "For help, run: \x1b[1mmoar --help\x1b[m")

		os.Exit(1)
	}

	for _, inputFilename := range flagSet.Args() {
		// Need to check before newScreen() below, otherwise the screen
		// will be cleared before we print the "No such file" error.
		err := tryOpen(inputFilename)
		if err != nil {
			return nil, nil, chroma.Style{}, nil, err
		}
	}

	if len(flagSet.Args()) == 0 && !stdinIsRedirected {
		fmt.Fprintln(os.Stderr, "ERROR: Filename or input pipe required")
		fmt.Fprintln(os.Stderr)
		printCommandline(os.Stderr)
		fmt.Fprintln(os.Stderr, "For help, run: \x1b[1mmoar --help\x1b[m")
		os.Exit(1)
	}

	if stdoutIsRedirected {
		err := pumpToStdout(flagSet.Args()...)
		if err != nil {
			return nil, nil, chroma.Style{}, nil, err
		}
		return nil, nil, chroma.Style{}, nil, nil
	}

	// INVARIANT: At this point, stdout is a terminal and we should proceed with
	// paging.
	stdoutIsTerminal := !stdoutIsRedirected
	if !stdoutIsTerminal {
		panic("Invariant broken: stdout is not a terminal")
	}

	if len(flagSet.Args()) > 1 {
		fmt.Fprintln(os.Stderr, "ERROR: Expected exactly one filename, or data piped from stdin")
		fmt.Fprintln(os.Stderr)
		printCommandline(os.Stderr)
		fmt.Fprintln(os.Stderr, "For help, run: \x1b[1mmoar --help\x1b[m")

		os.Exit(1)
	}

	formatter := formatters.TTY256
	if *terminalColorsCount == twin.ColorCount8 {
		formatter = formatters.TTY8
	} else if *terminalColorsCount == twin.ColorCount16 {
		formatter = formatters.TTY16
	} else if *terminalColorsCount == twin.ColorCount24bit {
		formatter = formatters.TTY16m
	}

	var reader *m.Reader
	shouldFormat := !*noReFormat
	if stdinIsRedirected {
		// Display input pipe contents
		reader = m.NewReaderFromStream("", os.Stdin, formatter, m.ReaderOptions{Lexer: *lexer, ShouldFormat: shouldFormat})
	} else {
		// Display the input file contents
		if len(flagSet.Args()) != 1 {
			panic("Invariant broken: Expected exactly one filename")
		}
		reader, err = m.NewReaderFromFilename(flagSet.Args()[0], formatter, m.ReaderOptions{Lexer: *lexer, ShouldFormat: shouldFormat})
		if err != nil {
			return nil, nil, chroma.Style{}, nil, err
		}
	}

	// If the user is doing "sudo something | moar" we can't show the UI until
	// we start getting data, otherwise we'll mess up sudo's password prompt.
	reader.AwaitFirstByte()

	// We got the first byte, this means sudo is done (if it was used) and we
	// can set up the UI.
	screen, err := newScreen(*mouseMode, *terminalColorsCount)
	if err != nil {
		// Ref: https://github.com/walles/moar/issues/149
		log.Debug("Failed to set up screen for paging, pumping to stdout instead: ", err)

		reader.PumpToStdout()

		return nil, nil, chroma.Style{}, nil, nil
	}

	var style chroma.Style = *styles.Get(defaultDarkTheme)
	if *styleOption == nil {
		t0 := time.Now()
		screen.RequestTerminalBackgroundColor()
		select {
		case event := <-screen.Events():
			// Event received, let's see if it's the one we want
			switch ev := event.(type) {

			case twin.EventTerminalBackgroundDetected:
				log.Debug("Terminal background color detected as ", ev.Color, " after ", time.Since(t0))

				distanceToBlack := ev.Color.Distance(twin.NewColor24Bit(0, 0, 0))
				distanceToWhite := ev.Color.Distance(twin.NewColor24Bit(255, 255, 255))
				if distanceToBlack < distanceToWhite {
					style = *styles.Get(defaultDarkTheme)
				} else {
					style = *styles.Get(defaultLightTheme)
				}

			default:
				log.Debug("Expected terminal background color event but got ", ev, " after ", time.Since(t0), " putting back and giving up")
				screen.Events() <- event
			}

		// The worst number I have measured was around 15ms, in GNOME Terminal
		// running inside of VirtualBox. 3x that should be enough for everyone
		// (TM).
		case <-time.After(50 * time.Millisecond):
			log.Debug("Terminal background color still not detected after ", time.Since(t0), ", giving up")
		}
	} else {
		style = **styleOption
	}
	log.Debug("Using style <", style.Name, ">")
	reader.SetStyleForHighlighting(style)

	pager := m.NewPager(reader)
	pager.WrapLongLines = *wrap
	pager.ShowLineNumbers = !*noLineNumbers
	pager.ShowStatusBar = !*noStatusBar
	pager.DeInit = !*noClearOnExit
	pager.QuitIfOneScreen = *quitIfOneScreen
	pager.StatusBarStyle = *statusBarStyle
	pager.UnprintableStyle = *unprintableStyle
	pager.ScrollLeftHint = *scrollLeftHint
	pager.ScrollRightHint = *scrollRightHint
	pager.SideScrollAmount = int(*shift)

	pager.TargetLineNumber = targetLineNumber
	if *follow && pager.TargetLineNumber == nil {
		reallyHigh := linenumbers.LineNumberMax()
		pager.TargetLineNumber = &reallyHigh
	}

	return pager, screen, style, &formatter, nil
}

func main() {
	var loglines strings.Builder
	log.SetOutput(&loglines)

	defer func() {
		err := recover()
		if len(loglines.String()) == 0 && err == nil {
			// No problems
			return
		}

		printProblemsHeader()

		if len(loglines.String()) > 0 {
			fmt.Fprintln(os.Stderr)
			// Consider not printing duplicate log messages more than once
			fmt.Fprintf(os.Stderr, "%s", loglines.String())
		}

		if err != nil {
			fmt.Fprintln(os.Stderr)
			panic(err)
		}

		os.Exit(1)
	}()

	stdinIsRedirected := !term.IsTerminal(int(os.Stdin.Fd()))
	stdoutIsRedirected := !term.IsTerminal(int(os.Stdout.Fd()))

	pager, screen, style, formatter, err := pagerFromArgs(
		os.Args,
		twin.NewScreenWithMouseModeAndColorCount,
		stdinIsRedirected,
		stdoutIsRedirected,
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR:", err)
		os.Exit(1)
	}

	if pager == nil {
		// No pager, we're done
		return
	}

	startPaging(pager, screen, &style, formatter)
}

// Define a generic flag with specified name, default value, and usage string.
// The return value is the address of a variable that stores the parsed value of
// the flag.
func flagSetFunc[T any](flagSet *flag.FlagSet, name string, defaultValue T, usage string, parser func(valueString string) (T, error)) *T {
	parsed := defaultValue

	flagSet.Func(name, usage, func(valueString string) error {
		parseResult, err := parser(valueString)
		if err != nil {
			return err
		}
		parsed = parseResult
		return nil
	})

	return &parsed
}

func startPaging(pager *m.Pager, screen twin.Screen, chromaStyle *chroma.Style, chromaFormatter *chroma.Formatter) {
	defer func() {
		// Restore screen...
		screen.Close()

		// ... before printing any panic() output, otherwise the output will
		// have broken linefeeds and be hard to follow.
		if err := recover(); err != nil {
			panic(err)
		}

		if !pager.DeInit {
			err := pager.ReprintAfterExit()
			if err != nil {
				log.Error("Failed reprinting pager view after exit: ", err)
			}
		}

		if pager.AfterExit != nil {
			err := pager.AfterExit()
			if err != nil {
				log.Error("Failed running AfterExit hook: ", err)
			}
		}
	}()

	pager.StartPaging(screen, chromaStyle, chromaFormatter)
}
