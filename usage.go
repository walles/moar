package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/walles/moar/m"
	"github.com/walles/moar/twin"
)

const flags_usage = `      --colors value              Highlighting palette size: 8, 16, 256, 16M, auto
      --debug                     Print debug logs after exiting
  -f, --follow                    Follow piped input just like "tail -f"
  -l, --lang value                File contents, used for highlighting. Mime type or extension ("html"). Default is to guess by filename.
      --mousemode mode            Mouse mode: auto, select or scroll: https://github.com/walles/moar/blob/master/MOUSE.md
      --no-clear-on-exit          Retain screen contents when exiting moar
      --no-linenumbers            Hide line numbers on startup, press left arrow key to show
      --no-reformat               No effect, kept for compatibility. See --reformat (default true)
      --no-statusbar              Hide the status bar, toggle with '='
  -F, --quit-if-one-screen        Don't page if contents fits on one screen
      --reformat                  Reformat some input files (JSON)
      --render-unprintable value  How unprintable characters are rendered: highlight or whitespace
      --scroll-left-hint value    Shown when view can scroll left. One character with optional ANSI highlighting.
      --scroll-right-hint value   Shown when view can scroll right. One character with optional ANSI highlighting.
      --shift amount              Horizontal scroll amount >= 1, defaults to 16
      --statusbar style           Status bar style: inverse, plain or bold
      --style style               Highlighting style from https://xyproto.github.io/splash/docs/longer/all.html
      --terminal-fg               Use terminal foreground color rather than style foreground for plain text
      --trace                     Print trace logs after exiting
  -w, --wrap                      Wrap long lines

  +1234                           Immediately scroll to line 1234

  -h, --help                      Show this help text
  -V, --version                   Print the moar version number`

func renderLessTermcapEnvVar(envVarName string, description string, colors twin.ColorCount) string {
	value := os.Getenv(envVarName)
	if len(value) == 0 {
		return ""
	}

	style, err := m.TermcapToStyle(value)
	if err != nil {
		bold := twin.StyleDefault.WithAttr(twin.AttrBold).RenderUpdateFrom(twin.StyleDefault, colors)
		notBold := twin.StyleDefault.RenderUpdateFrom(twin.StyleDefault.WithAttr(twin.AttrBold), colors)
		return fmt.Sprintf("  %s (%s): %s %s<- Error: %v%s\n",
			envVarName,
			description,
			strings.ReplaceAll(value, "\x1b", "ESC"),
			bold,
			err,
			notBold,
		)
	}

	prefix := style.RenderUpdateFrom(twin.StyleDefault, colors)
	suffix := twin.StyleDefault.RenderUpdateFrom(style, colors)
	return fmt.Sprintf("  %s (%s): %s\n",
		envVarName,
		description,
		prefix+strings.ReplaceAll(value, "\x1b", "ESC")+suffix,
	)
}

func renderPagerEnvVar(name string, colors twin.ColorCount) string {
	bold := twin.StyleDefault.WithAttr(twin.AttrBold).RenderUpdateFrom(twin.StyleDefault, colors)
	notBold := twin.StyleDefault.RenderUpdateFrom(twin.StyleDefault.WithAttr(twin.AttrBold), colors)

	value, isSet := os.LookupEnv(name)
	if value == "" {
		what := "unset"
		if isSet {
			what = "empty"
		}

		return fmt.Sprintf("  %s is %s %s<- Should be %s%s\n",
			name,
			what,
			bold,
			getMoarPath(),
			notBold,
		)
	}

	absMoarPath, err := absLookPath(os.Args[0])
	if err != nil {
		log.Warn("Unable to find absolute moar path: ", err)
		return ""
	}

	absEnvValue, err := absLookPath(value)
	if err != nil {
		// This can happen if this is set to some outdated value
		absEnvValue = value
	}

	if absEnvValue == absMoarPath {
		return fmt.Sprintf("  %s=%s\n", name, value)
	}

	return fmt.Sprintf("  %s=%s %s<- Should be %s%s\n",
		name,
		value,
		bold,
		getMoarPath(),
		notBold,
	)
}

// If the environment variable is set, render it as APA=bepa indented two
// spaces, plus a newline at the end. Otherwise, return an empty string.
func renderPlainEnvVar(envVarName string) string {
	value := os.Getenv(envVarName)
	if value == "" {
		return ""
	}

	return fmt.Sprintf("  %s=%s\n", envVarName, value)
}

func printCommandline(output io.Writer) {
	fmt.Fprintln(output, "Commandline: moar", strings.Join(os.Args[1:], " "))
	fmt.Fprintf(output, "Environment: MOAR=\"%v\"\n", os.Getenv("MOAR"))
	fmt.Fprintln(output)
}

func heading(text string, colors twin.ColorCount) string {
	style := twin.StyleDefault.WithAttr(twin.AttrItalic)
	prefix := style.RenderUpdateFrom(twin.StyleDefault, colors)
	suffix := twin.StyleDefault.RenderUpdateFrom(style, colors)
	return prefix + text + suffix
}

func printUsage(flagSet *flag.FlagSet, colors twin.ColorCount) {
	// This controls where PrintDefaults() prints, see below
	flagSet.SetOutput(os.Stdout)

	// FIXME: Log if any printouts fail?

	fmt.Println(heading("Usage", colors))
	fmt.Println("  moar [options] <file>")
	fmt.Println("  ... | moar")
	fmt.Println("  moar < file")
	fmt.Println()
	fmt.Println("Shows file contents. Compressed files will be transparently decompressed.")
	fmt.Println("Input is expected to be (possibly compressed) UTF-8 encoded text. Invalid /")
	fmt.Println("non-printable characters are by default rendered as '?'.")
	fmt.Println()
	fmt.Println("More information + source code:")
	fmt.Println("  <https://github.com/walles/moar#readme>")
	fmt.Println()
	fmt.Println(heading("Environment", colors))

	moarEnv := os.Getenv("MOAR")
	if len(moarEnv) == 0 {
		fmt.Println("  Additional options are read from the MOAR environment variable if set.")
		fmt.Println("  But currently, the MOAR environment variable is not set.")
	} else {
		fmt.Println("  Additional options are read from the MOAR environment variable.")
		fmt.Printf("  Current setting: MOAR=\"%s\"\n", moarEnv)
	}

	envSection := ""
	envSection += renderLessTermcapEnvVar("LESS_TERMCAP_md", "man page bold style", colors)
	envSection += renderLessTermcapEnvVar("LESS_TERMCAP_us", "man page underline style", colors)
	envSection += renderLessTermcapEnvVar("LESS_TERMCAP_so", "search hits and footer style", colors)

	envSection += renderPagerEnvVar("PAGER", colors)
	envVars := os.Environ()
	sort.Strings(envVars)
	for _, env := range envVars {
		split := strings.SplitN(env, "=", 2)
		if len(split) != 2 {
			continue
		}

		name := split[0]
		if name == "PAGER" {
			// Already done above
			continue
		}
		if !strings.HasSuffix(name, "PAGER") {
			continue
		}

		envSection += renderPagerEnvVar(name, colors)
	}

	envSection += renderPlainEnvVar("TERM")
	envSection += renderPlainEnvVar("TERM_PROGRAM")
	envSection += renderPlainEnvVar("COLORTERM")

	// Requested here: https://github.com/walles/moar/issues/170#issuecomment-1891154661
	envSection += renderPlainEnvVar("MANROFFOPT")

	if envSection != "" {
		fmt.Println()

		// Not Println since the section already ends with a newline
		fmt.Print(envSection)
	}

	printSetDefaultPagerHelp(colors)

	fmt.Println()
	fmt.Println(heading("Options", colors))

	fmt.Println(flags_usage)
}

// If $PAGER isn't pointing to us, print a help text on how to set it.
func printSetDefaultPagerHelp(colors twin.ColorCount) {
	absMoarPath, err := absLookPath(os.Args[0])
	if err != nil {
		log.Warn("Unable to find moar binary ", err)
		return
	}

	absPagerValue, err := absLookPath(os.Getenv("PAGER"))
	if err != nil {
		absPagerValue = ""
	}

	if absPagerValue == absMoarPath {
		// We're already the default pager
		return
	}

	fmt.Println()
	fmt.Println(heading("Making moar Your Default Pager", colors))

	shellIsFish := strings.HasSuffix(os.Getenv("SHELL"), "fish")
	shellIsPowershell := len(os.Getenv("PSModulePath")) > 0

	if shellIsFish {
		fmt.Println("  Write this command at your prompt:")
		fmt.Println()
		fmt.Printf("     set -Ux PAGER %s\n", getMoarPath())
	} else if shellIsPowershell {
		fmt.Println("  Put the following line in your $PROFILE file (\"echo $PROFILE\" to find it)")
		fmt.Println("  and moar will be used as the default pager in all new terminal windows:")
		fmt.Println()
		fmt.Printf("     $env:PAGER = \"%s\"\n", getMoarPath())
	} else {
		// I don't know how to identify bash / zsh, put generic instructions here
		fmt.Println("  Put the following line in your ~/.bashrc, ~/.bash_profile or ~/.zshrc")
		fmt.Println("  and moar will be used as the default pager in all new terminal windows:")
		fmt.Println()
		fmt.Printf("     export PAGER=%s\n", getMoarPath())
	}
}

// "moar" if we're in the $PATH, otherwise an absolute path
func getMoarPath() string {
	moarPath := os.Args[0]
	if filepath.IsAbs(moarPath) {
		return moarPath
	}

	if strings.Contains(moarPath, string(os.PathSeparator)) {
		// Relative path
		moarPath, err := filepath.Abs(moarPath)
		if err != nil {
			panic(err)
		}
		return moarPath
	}

	// Neither absolute nor relative, try PATH
	_, err := exec.LookPath(moarPath)
	if err != nil {
		panic("Unable to find in $PATH: " + moarPath)
	}
	return moarPath
}

func absLookPath(path string) (string, error) {
	lookedPath, err := exec.LookPath(path)
	if err != nil {
		return "", err
	}

	absLookedPath, err := filepath.Abs(lookedPath)
	if err != nil {
		return "", err
	}

	return absLookedPath, err
}
