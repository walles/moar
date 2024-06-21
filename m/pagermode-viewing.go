package m

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	log "github.com/sirupsen/logrus"
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
	editorStat, err := os.Stat(editorPath)
	if err != nil {
		// FIXME: Show a message in the status bar instead? Nothing wrong with
		// moar here.
		log.Warn("Failed to stat editor "+editorPath+": ", err)
		return
	}
	if editorStat.Mode()&0111 == 0 {
		// Note that this check isn't perfect, it could still be executable but
		// not by us. Corner case, let's just complain about that when trying to
		// run it.

		// FIXME: Show a message in the status bar instead? Nothing wrong with
		// moar here.
		log.Warn("Editor " + editorPath + " is not executable")
		return
	}

	var fileToEdit string
	if p.reader.fileName != nil {
		fileToEdit = *p.reader.fileName
	} else {
		// FIXME: If the buffer is from stdin, store it in a temp file. Consider
		// naming it based on the current language setting.

		// FIXME: Should we wait for the stream to end before launching the
		// editor? Maybe no?
		panic("Make a temp file with the buffer contents")
	}

	// Verify that the file exists and is readable
	fileToEditStat, err := os.Stat(fileToEdit)
	if err != nil {
		log.Warn("Failed to stat file to edit "+fileToEdit+": ", err)
		return
	}
	if fileToEditStat.Mode()&0444 == 0 {
		log.Warn("File to edit " + fileToEdit + " is not readable")
		return
	}

	p.AfterExit = func() error {
		// NOTE: If you do any changes here, make sure they work with both nano
		// and "code -w".
		commandWithArgs := strings.Fields(editor)
		commandWithArgs = append(commandWithArgs, fileToEdit)

		// syscall.Exec() will terminate ourselves if it succeeds
		err := syscall.Exec(editorPath, commandWithArgs, os.Environ())

		// When we get here, err will always be non-nil
		return fmt.Errorf("Failed to exec %s %s: %w", editorPath, commandWithArgs[1:], err)
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
