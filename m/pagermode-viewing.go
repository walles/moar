package m

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/walles/moar/m/linenumbers"
	"github.com/walles/moar/twin"
)

type PagerModeViewing struct {
	pager *Pager
}

func (m PagerModeViewing) drawFooter(statusText string, spinner string) {
	helpText := "Press 'ESC' / 'q' to exit, '/' to search, '?' for help"
	if m.pager.isShowingHelp {
		helpText = "Press 'ESC' / 'q' to exit help, '/' to search"
	}

	if m.pager.ShowStatusBar {
		m.pager.setFooter(statusText + spinner + "  " + helpText)
	}
}

func (m PagerModeViewing) onKey(keyCode twin.KeyCode) {
	p := m.pager

	switch keyCode {
	case twin.KeyEscape:
		p.Quit()

	case twin.KeyUp:
		// Clipping is done in _Redraw()
		p.scrollPosition = p.scrollPosition.PreviousLine(1)
		p.handleScrolledUp()

	case twin.KeyDown, twin.KeyEnter:
		// Clipping is done in _Redraw()
		p.scrollPosition = p.scrollPosition.NextLine(1)
		p.handleScrolledDown()

	case twin.KeyRight:
		p.moveRight(p.SideScrollAmount)

	case twin.KeyLeft:
		p.moveRight(-p.SideScrollAmount)

	case twin.KeyAltRight:
		p.moveRight(1)

	case twin.KeyAltLeft:
		p.moveRight(-1)

	case twin.KeyHome:
		p.scrollPosition = newScrollPosition("Pager scroll position")
		p.handleScrolledUp()

	case twin.KeyEnd:
		p.scrollToEnd()

	case twin.KeyPgUp:
		p.scrollPosition = p.scrollPosition.PreviousLine(p.visibleHeight())
		p.handleScrolledUp()

	case twin.KeyPgDown:
		p.scrollPosition = p.scrollPosition.NextLine(p.visibleHeight())
		p.handleScrolledDown()

	default:
		log.Debugf("Unhandled key event %v", keyCode)
	}
}

// Dump the reader lines into a read-only temp file and return the absolute file
// name.
func dumpToTempFile(reader *Reader) (string, error) {
	tempFile, err := os.CreateTemp("", "moar-contents-")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	log.Debug("Dumping contents into: ", tempFile.Name())

	lines, _ := reader.GetLines(linenumbers.LineNumber{}, math.MaxInt)
	for _, line := range lines.lines {
		_, err := tempFile.WriteString(line.raw + "\n")
		if err != nil {
			return "", err
		}
	}

	// Ref: https://pkg.go.dev/os#Chmod
	err = os.Chmod(tempFile.Name(), 0400)
	if err != nil {
		// Doesn't matter that much, but if it fails we should at least log it
		log.Debug("Failed to make temp file ", tempFile.Name(), " read-only: ", err)
	}

	return tempFile.Name(), nil
}

// Check that the editor is executable
func errUnlessExecutable(file string) error {
	stat, err := os.Stat(file)
	if err != nil {
		return fmt.Errorf("Failed to stat %s: %w", file, err)
	}
	if stat.Mode()&0111 == 0 {
		// Note that this check isn't perfect, it could still be executable but
		// not by us. Corner case, let's just fail later in that case.
		return fmt.Errorf("Not executable: %s", file)
	}

	return nil
}

func handleEditingRequest(p *Pager) {
	// Get an editor setting from either VISUAL or EDITOR
	editorEnv := "VISUAL"
	editor := strings.TrimSpace(os.Getenv(editorEnv))
	if editor == "" {
		editorEnv := "EDITOR"
		editor = strings.TrimSpace(os.Getenv(editorEnv))
	}
	if editor == "" {
		// FIXME: Show a message in the status bar instead? Nothing wrong with
		// moar here.
		log.Warn("Neither $VISUAL nor $EDITOR are set, can't launch any editor")
		return
	}

	// Tyre kicking check that we can find the editor either in the PATH or as
	// an absolute path
	firstWord := strings.Fields(editor)[0]
	editorPath, err := exec.LookPath(firstWord)
	if err != nil {
		// FIXME: Show a message in the status bar instead? Nothing wrong with
		// moar here.
		log.Warn("Failed to find editor "+firstWord+" from $"+editorEnv+": ", err)
		return
	}

	// Check that the editor is executable
	err = errUnlessExecutable(editorPath)
	if err != nil {
		// FIXME: Show a message in the status bar instead? Nothing wrong with
		// moar here.
		log.Warn("Editor not executable: {}", err)
		return
	}

	canOpenFile := p.reader.fileName != nil
	if p.reader.fileName != nil {
		// Verify that the file exists and is readable
		err = tryOpen(*p.reader.fileName)
		if err != nil {
			canOpenFile = false
			log.Info("File to edit is not readable: ", err)
		}
	}

	var fileToEdit string
	if canOpenFile {
		fileToEdit = *p.reader.fileName
	} else {
		// NOTE: Let's not wait for the stream to finish, just dump whatever we
		// have and open the editor on that. The user just asked for it, if they
		// wanted to wait, they should have done that themselves.

		// Create a temp file based on reader contents
		fileToEdit, err = dumpToTempFile(p.reader)
		if err != nil {
			log.Warn("Failed to create temp file to edit: ", err)
			return
		}
	}

	p.AfterExit = func() error {
		// NOTE: If you do any changes here, make sure they work with both "nano"
		// and "code -w" (VSCode).
		commandWithArgs := strings.Fields(editor)
		commandWithArgs = append(commandWithArgs, fileToEdit)

		log.Info("'v' pressed, launching editor: ", commandWithArgs)
		command := exec.Command(commandWithArgs[0], commandWithArgs[1:]...)

		// Since os.Stdin might come from a pipe, we can't trust that. Instead,
		// we tell the editor to read from os.Stdout, which points to the
		// terminal as well.
		//
		// Tested on macOS and Linux, works like a charm.
		command.Stdin = os.Stdout // <- YES, WE SHOULD ASSIGN STDOUT TO STDIN

		command.Stdout = os.Stdout
		command.Stderr = os.Stderr

		err := command.Run()
		if err == nil {
			log.Info("Editor exited successfully: ", commandWithArgs)
		}
		return err
	}
	p.Quit()
}

func (m PagerModeViewing) onRune(char rune) {
	p := m.pager

	switch char {
	case 'q':
		p.Quit()

	case 'v':
		handleEditingRequest(p)

	case '?':
		if !p.isShowingHelp {
			p.preHelpState = &_PreHelpState{
				reader:              p.reader,
				scrollPosition:      p.scrollPosition,
				leftColumnZeroBased: p.leftColumnZeroBased,
				targetLineNumber:    p.TargetLineNumber,
			}
			p.reader = _HelpReader
			p.scrollPosition = newScrollPosition("Pager scroll position")
			p.leftColumnZeroBased = 0
			p.TargetLineNumber = nil
			p.isShowingHelp = true
		}

	case '=':
		p.ShowStatusBar = !p.ShowStatusBar

	// '\x10' = CTRL-p, should scroll up one line.
	// Ref: https://github.com/walles/moar/issues/107#issuecomment-1328354080
	case 'k', 'y', '\x10':
		// Clipping is done in _Redraw()
		p.scrollPosition = p.scrollPosition.PreviousLine(1)
		p.handleScrolledUp()

	// '\x0e' = CTRL-n, should scroll down one line.
	// Ref: https://github.com/walles/moar/issues/107#issuecomment-1328354080
	case 'j', 'e', '\x0e':
		// Clipping is done in _Redraw()
		p.scrollPosition = p.scrollPosition.NextLine(1)
		p.handleScrolledDown()

	case 'l':
		// vim right
		p.moveRight(p.SideScrollAmount)

	case 'h':
		// vim left
		p.moveRight(-p.SideScrollAmount)

	case '<':
		p.scrollPosition = newScrollPosition("Pager scroll position")
		p.handleScrolledUp()

	case '>', 'G':
		p.scrollToEnd()

	case 'f', ' ':
		p.scrollPosition = p.scrollPosition.NextLine(p.visibleHeight())
		p.handleScrolledDown()

	case 'b':
		p.scrollPosition = p.scrollPosition.PreviousLine(p.visibleHeight())
		p.handleScrolledUp()

	// '\x15' = CTRL-u, should work like just 'u'.
	// Ref: https://github.com/walles/moar/issues/90
	case 'u', '\x15':
		p.scrollPosition = p.scrollPosition.PreviousLine(p.visibleHeight() / 2)
		p.handleScrolledUp()

	// '\x04' = CTRL-d, should work like just 'd'.
	// Ref: https://github.com/walles/moar/issues/90
	case 'd', '\x04':
		p.scrollPosition = p.scrollPosition.NextLine(p.visibleHeight() / 2)
		p.handleScrolledDown()

	case '/':
		p.mode = PagerModeSearch{pager: p}
		p.TargetLineNumber = nil
		p.searchString = ""
		p.searchPattern = nil

	case 'g':
		p.mode = &PagerModeGotoLine{pager: p}
		p.TargetLineNumber = nil

	// Should match the pagermode-not-found.go previous-search-hit bindings
	case 'n':
		p.scrollToNextSearchHit()

	// Should match the pagermode-not-found.go next-search-hit bindings
	case 'p', 'N':
		p.scrollToPreviousSearchHit()

	case 'm':
		p.mode = PagerModeMark{pager: p}
		p.TargetLineNumber = nil

	case '\'':
		p.mode = PagerModeJumpToMark{pager: p}
		p.TargetLineNumber = nil

	case 'w':
		p.WrapLongLines = !p.WrapLongLines

	default:
		log.Debugf("Unhandled rune keypress '%s'/0x%08x", string(char), int32(char))
	}
}
