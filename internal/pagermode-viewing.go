package internal

import (
	log "github.com/sirupsen/logrus"
	"github.com/walles/moar/twin"
)

type PagerModeViewing struct {
	pager *Pager
}

func (m PagerModeViewing) drawFooter(statusText string, spinner string) {
	helpText := "Press 'ESC' / 'q' to exit, '/' to search, '&' to filter, 'h' for help"
	if m.pager.isShowingHelp {
		helpText = "Press 'ESC' / 'q' to exit help, '/' to search"
	}

	if m.pager.ShowStatusBar {
		if len(spinner) > 0 {
			spinner = "  " + spinner
		}
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

func (m PagerModeViewing) onRune(char rune) {
	p := m.pager

	switch char {
	case 'q':
		p.Quit()

	case 'v':
		handleEditingRequest(p)

	case 'h':
		if p.isShowingHelp {
			break
		}

		p.preHelpState = &_PreHelpState{
			scrollPosition:      p.scrollPosition,
			leftColumnZeroBased: p.leftColumnZeroBased,
			targetLine:          p.TargetLine,
		}
		p.scrollPosition = newScrollPosition("Pager scroll position")
		p.leftColumnZeroBased = 0
		p.setTargetLine(nil)
		p.isShowingHelp = true

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
		p.mode = PagerModeSearch{pager: p, direction: SearchDirectionForward, initialScrollPosition: p.scrollPosition}
		p.setTargetLine(nil)
		p.searchString = ""
		p.searchPattern = nil

	case '?':
		p.mode = PagerModeSearch{pager: p, direction: SearchDirectionBackward, initialScrollPosition: p.scrollPosition}
		p.setTargetLine(nil)
		p.searchString = ""
		p.searchPattern = nil

	case '&':
		if !p.isShowingHelp {
			// Filtering the help text is not supported. Feel free to work on
			// that if you feel that's time well spent.
			p.mode = &PagerModeFilter{pager: p}
			p.searchString = ""
			p.searchPattern = nil
			p.filterPattern = nil
		}

	case 'g':
		p.mode = &PagerModeGotoLine{pager: p}
		p.setTargetLine(nil)

	// Should match the pagermode-not-found.go previous-search-hit bindings
	case 'n':
		p.scrollToNextSearchHit()

	// Should match the pagermode-not-found.go next-search-hit bindings
	case 'p', 'N':
		p.scrollToPreviousSearchHit()

	case 'm':
		p.mode = PagerModeMark{pager: p}
		p.setTargetLine(nil)

	case '\'':
		p.mode = PagerModeJumpToMark{pager: p}
		p.setTargetLine(nil)

	case 'w':
		p.WrapLongLines = !p.WrapLongLines

	default:
		log.Debugf("Unhandled rune keypress '%s'/0x%08x", string(char), int32(char))
	}
}
